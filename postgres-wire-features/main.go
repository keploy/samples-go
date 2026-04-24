package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type pgClient struct {
	conn net.Conn
	user string
}

type queryResult struct {
	SQL      string     `json:"sql"`
	Commands []string   `json:"commands"`
	Rows     [][]string `json:"rows"`
	CopyData []string   `json:"copyData,omitempty"`
}

type caseResult struct {
	Name   string        `json:"name"`
	Error  string        `json:"error,omitempty"`
	Result []queryResult `json:"result,omitempty"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	cases := map[string]func() ([]queryResult, error){
		"setup":           runSetup,
		"dml":             runDML,
		"catalog-cte":     runCatalogCTE,
		"catalog-sub":     runCatalogSubselect,
		"catalog-setop":   runCatalogSetOp,
		"copy":            runCopy,
		"prepare":         runPrepareExecute,
		"cursor":          runCursor,
		"admin":           runAdminStatements,
		"validation-ping": runValidationPing,
	}

	for name, fn := range cases {
		name, fn := name, fn
		mux.HandleFunc("/case/"+name, func(w http.ResponseWriter, _ *http.Request) {
			writeCase(w, name, fn)
		})
	}

	mux.HandleFunc("/run/all", func(w http.ResponseWriter, _ *http.Request) {
		order := []string{
			"setup",
			"dml",
			"catalog-cte",
			"catalog-sub",
			"catalog-setop",
			"copy",
			"prepare",
			"cursor",
			"admin",
			"validation-ping",
		}
		results := make([]caseResult, 0, len(order))
		status := http.StatusOK
		for _, name := range order {
			res, err := cases[name]()
			item := caseResult{Name: name, Result: res}
			if err != nil {
				item.Error = err.Error()
				status = http.StatusInternalServerError
			}
			results = append(results, item)
		}
		writeJSON(w, status, results)
	})

	addr := env("ADDR", ":8080")
	log.Printf("postgres-wire-features listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeCase(w http.ResponseWriter, name string, fn func() ([]queryResult, error)) {
	res, err := fn()
	status := http.StatusOK
	item := caseResult{Name: name, Result: res}
	if err != nil {
		status = http.StatusInternalServerError
		item.Error = err.Error()
	}
	writeJSON(w, status, item)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func withClient(fn func(*pgClient) ([]queryResult, error)) ([]queryResult, error) {
	c, err := dialPostgres()
	if err != nil {
		return nil, err
	}
	defer c.conn.Close()
	return fn(c)
}

func runSetup() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`DROP TABLE IF EXISTS gap_items`,
			`CREATE TABLE gap_items (id SERIAL PRIMARY KEY, name TEXT NOT NULL)`,
			`INSERT INTO gap_items(name) VALUES ('seed-a'), ('seed-b')`,
			`CREATE TABLE IF NOT EXISTS flyway_schema_history (
				installed_rank INT PRIMARY KEY,
				version TEXT,
				description TEXT,
				type TEXT,
				script TEXT,
				checksum INT,
				installed_by TEXT,
				installed_on TIMESTAMP DEFAULT now(),
				execution_time INT,
				success BOOLEAN
			)`,
		)
	})
}

func runDML() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`INSERT INTO flyway_schema_history
				(installed_rank, version, description, type, script, checksum, installed_by, execution_time, success)
			 VALUES (1, '1', 'v3 dml classifier repro', 'SQL', 'V1__gap.sql', 42, 'gap-app', 7, true)
			 ON CONFLICT (installed_rank) DO UPDATE SET checksum = EXCLUDED.checksum`,
			`UPDATE gap_items SET name = name || '-updated' WHERE id = 1`,
			`DELETE FROM gap_items WHERE name = 'never-present'`,
			`MERGE INTO gap_items AS target
			 USING (VALUES (2, 'seed-b-merged')) AS src(id, name)
			 ON target.id = src.id
			 WHEN MATCHED THEN UPDATE SET name = src.name
			 WHEN NOT MATCHED THEN INSERT (id, name) VALUES (src.id, src.name)`,
		)
	})
}

func runCatalogCTE() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`WITH catalog_rels AS (
				SELECT oid, relname FROM pg_catalog.pg_class WHERE relname IN ('pg_class', 'pg_type')
			 )
			 SELECT relname FROM catalog_rels ORDER BY relname`,
		)
	})
}

func runCatalogSubselect() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`SELECT relname
			 FROM (SELECT relname FROM pg_catalog.pg_class WHERE relname = 'pg_class') AS catalog_subquery`,
		)
	})
}

func runCatalogSetOp() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`SELECT relname FROM pg_catalog.pg_class WHERE relname = 'pg_class'
			 UNION
			 SELECT relname FROM pg_catalog.pg_class WHERE relname = 'pg_type'
			 ORDER BY relname`,
		)
	})
}

func runCopy() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		var all []queryResult
		setup, err := runQueries(c, `TRUNCATE gap_items RESTART IDENTITY`)
		if err != nil {
			return all, err
		}
		all = append(all, setup...)

		in, err := c.simpleQuery(`COPY gap_items(name) FROM STDIN`, "copy-a\ncopy-b\n")
		if err != nil {
			return all, err
		}
		all = append(all, in)

		out, err := c.simpleQuery(`COPY (SELECT id, name FROM gap_items ORDER BY id) TO STDOUT`, "")
		if err != nil {
			return all, err
		}
		all = append(all, out)
		return all, nil
	})
}

func runPrepareExecute() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`PREPARE gap_lookup(int) AS SELECT name FROM gap_items WHERE id = $1`,
			`EXECUTE gap_lookup(1)`,
			`DEALLOCATE gap_lookup`,
		)
	})
}

func runCursor() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`BEGIN`,
			`DECLARE gap_cursor CURSOR FOR SELECT id, name FROM gap_items ORDER BY id`,
			`FETCH 1 FROM gap_cursor`,
			`FETCH 1 FROM gap_cursor`,
			`CLOSE gap_cursor`,
			`COMMIT`,
		)
	})
}

func runAdminStatements() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`ANALYZE gap_items`,
			`REINDEX TABLE gap_items`,
			`BEGIN`,
			`LOCK TABLE gap_items IN ACCESS SHARE MODE`,
			`COMMIT`,
			`DO $$ BEGIN PERFORM 1; END $$`,
			`CREATE OR REPLACE PROCEDURE gap_noop() LANGUAGE plpgsql AS $$ BEGIN PERFORM 1; END $$`,
			`CALL gap_noop()`,
		)
	})
}

func runValidationPing() ([]queryResult, error) {
	return withClient(func(c *pgClient) ([]queryResult, error) {
		return runQueries(c,
			`SELECT 'ok'`,
			`SELECT true`,
			`SELECT NULL`,
			`SELECT 1`,
		)
	})
}

func runQueries(c *pgClient, queries ...string) ([]queryResult, error) {
	results := make([]queryResult, 0, len(queries))
	for _, q := range queries {
		res, err := c.simpleQuery(q, "")
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

func dialPostgres() (*pgClient, error) {
	host := env("PGHOST", "127.0.0.1")
	port := env("PGPORT", "5432")
	user := env("PGUSER", "postgres")
	db := env("PGDATABASE", "postgres")
	password := env("PGPASSWORD", "")

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		return nil, err
	}

	c := &pgClient{conn: conn, user: user}
	if err := c.startup(user, db, password); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *pgClient) startup(user, db, password string) error {
	var body bytes.Buffer
	_ = binary.Write(&body, binary.BigEndian, int32(196608))
	writeCString(&body, "user")
	writeCString(&body, user)
	writeCString(&body, "database")
	writeCString(&body, db)
	writeCString(&body, "client_encoding")
	writeCString(&body, "UTF8")
	body.WriteByte(0)

	msg := withLength(body.Bytes())
	if _, err := c.conn.Write(msg); err != nil {
		return err
	}

	for {
		tag, payload, err := c.readMessage()
		if err != nil {
			return err
		}
		switch tag {
		case 'R':
			if len(payload) < 4 {
				return errors.New("short authentication message")
			}
			code := binary.BigEndian.Uint32(payload[:4])
			switch code {
			case 0:
			case 3:
				if password == "" {
					return errors.New("postgres requested cleartext password, but PGPASSWORD is empty")
				}
				if err := c.sendPassword(password); err != nil {
					return err
				}
			case 5:
				if password == "" {
					return errors.New("postgres requested md5 password, but PGPASSWORD is empty")
				}
				if len(payload) < 8 {
					return errors.New("short md5 authentication salt")
				}
				if err := c.sendPassword(md5Password(user, password, payload[4:8])); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported postgres authentication code %d", code)
			}
		case 'S', 'K':
		case 'Z':
			return nil
		case 'E':
			return errors.New(parseError(payload))
		default:
			return fmt.Errorf("unexpected startup message %q", tag)
		}
	}
}

func (c *pgClient) sendPassword(password string) error {
	var body bytes.Buffer
	writeCString(&body, password)
	return c.writeMessage('p', body.Bytes())
}

func (c *pgClient) simpleQuery(sql, copyInput string) (queryResult, error) {
	res := queryResult{SQL: oneLine(sql)}
	var body bytes.Buffer
	writeCString(&body, sql)
	if err := c.writeMessage('Q', body.Bytes()); err != nil {
		return res, err
	}

	for {
		tag, payload, err := c.readMessage()
		if err != nil {
			return res, err
		}
		switch tag {
		case 'T':
			// Row descriptions are not needed for this repro; DataRow carries values.
		case 'D':
			res.Rows = append(res.Rows, parseDataRow(payload))
		case 'C':
			res.Commands = append(res.Commands, trimCString(payload))
		case 'G':
			if copyInput == "" {
				return res, errors.New("server requested COPY input but no copy data was provided")
			}
			if err := c.writeMessage('d', []byte(copyInput)); err != nil {
				return res, err
			}
			if err := c.writeMessage('c', nil); err != nil {
				return res, err
			}
		case 'H':
			// COPY OUT starts; data arrives in CopyData messages.
		case 'd':
			res.CopyData = append(res.CopyData, strings.TrimRight(string(payload), "\n"))
		case 'c':
			// CopyDone from server after COPY TO STDOUT.
		case 'N':
		case 'n', 's':
		case 'Z':
			return res, nil
		case 'E':
			return res, errors.New(parseError(payload))
		default:
			return res, fmt.Errorf("unhandled postgres message %q for query %q", tag, oneLine(sql))
		}
	}
}

func (c *pgClient) readMessage() (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return 0, nil, err
	}
	size := int(binary.BigEndian.Uint32(header[1:5]))
	if size < 4 {
		return 0, nil, fmt.Errorf("invalid postgres message length %d", size)
	}
	payload := make([]byte, size-4)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return 0, nil, err
	}
	return header[0], payload, nil
}

func (c *pgClient) writeMessage(tag byte, payload []byte) error {
	msg := make([]byte, 1, 1+4+len(payload))
	msg[0] = tag
	msg = append(msg, withLength(payload)...)
	_, err := c.conn.Write(msg)
	return err
}

func withLength(payload []byte) []byte {
	msg := make([]byte, 4, 4+len(payload))
	binary.BigEndian.PutUint32(msg[:4], uint32(len(payload)+4))
	return append(msg, payload...)
}

func writeCString(buf *bytes.Buffer, value string) {
	buf.WriteString(value)
	buf.WriteByte(0)
}

func parseDataRow(payload []byte) []string {
	if len(payload) < 2 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(payload[:2]))
	pos := 2
	row := make([]string, 0, n)
	for i := 0; i < n && pos+4 <= len(payload); i++ {
		size := int(int32(binary.BigEndian.Uint32(payload[pos : pos+4])))
		pos += 4
		if size == -1 {
			row = append(row, "NULL")
			continue
		}
		if pos+size > len(payload) {
			row = append(row, "<short-value>")
			break
		}
		row = append(row, string(payload[pos:pos+size]))
		pos += size
	}
	return row
}

func parseError(payload []byte) string {
	parts := make([]string, 0, 4)
	for len(payload) > 0 && payload[0] != 0 {
		fieldType := payload[0]
		payload = payload[1:]
		idx := bytes.IndexByte(payload, 0)
		if idx < 0 {
			break
		}
		value := string(payload[:idx])
		payload = payload[idx+1:]
		if fieldType == 'S' || fieldType == 'C' || fieldType == 'M' {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return "postgres error"
	}
	return strings.Join(parts, ": ")
}

func trimCString(payload []byte) string {
	return strings.TrimRight(string(payload), "\x00")
}

func oneLine(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func md5Password(user, password string, salt []byte) string {
	first := md5.Sum([]byte(password + user))
	firstHex := make([]byte, hex.EncodedLen(len(first)))
	hex.Encode(firstHex, first[:])
	secondInput := append(firstHex, salt...)
	second := md5.Sum(secondInput)
	return "md5" + hex.EncodeToString(second[:])
}
