package uss

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	sql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MetaStore is a global instance of the Store.
var MetaStore *Store

// ShortCodeInfo defines the database model for a shortened URL.
type ShortCodeInfo struct {
	UID       uint64    `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	ShortCode string    `json:"shortcode" gorm:"uniqueIndex"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime(6);autoUpdateTime"`
	EndTime   time.Time `json:"end_time"   gorm:"type:datetime(6)"`
	CreatedBy string    `json:"created_by"`
}

// Store manages the database connection and operations.
type Store struct {
	db *gorm.DB
}

func registerTLSConfig(config map[string]string) error {
	if sslMode, exists := config["MYSQL_SSL_MODE"]; exists && sslMode == "production" {
		if caPath, exists := config["MYSQL_SSL_CA"]; exists && caPath != "" {
			rootCertPool := x509.NewCertPool()
			pem, err := os.ReadFile(caPath)
			if err != nil {
				return fmt.Errorf("failed to read CA file: %w", err)
			}
			if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
				return fmt.Errorf("failed to append CA certs")
			}

			tlsConfig := &tls.Config{
				RootCAs:            rootCertPool,
				InsecureSkipVerify: true,
			}

			if err := sql.RegisterTLSConfig(sslMode, tlsConfig); err != nil {
				return fmt.Errorf("failed to register TLS config '%s': %w", sslMode, err)
			}
			return nil
		}
	}
	return nil
}

func (s *Store) Connect(config map[string]string) error {
	if err := registerTLSConfig(config); err != nil {
		return fmt.Errorf("failed to register TLS config: %w", err)
	}

	var err error
	sslMode := config["MYSQL_SSL_MODE"]
	if sslMode == "" {
		sslMode = "false"
	}
	mysqlDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local&tls=%s",
		config["MYSQL_USER"],
		config["MYSQL_PASSWORD"],
		config["MYSQL_HOST"],
		config["MYSQL_PORT"],
		config["MYSQL_DBNAME"],
		sslMode,
	)
	s.db, err = gorm.Open(mysql.New(mysql.Config{
		DSN:               mysqlDSN,
		DefaultStringSize: 256,
	}), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Only enforce SSL verification if the mode is set to 'production'
	if config["MYSQL_SSL_MODE"] == "production" {
		var sslStatus string
		var variableName string
		err := s.db.Raw("SHOW STATUS LIKE 'Ssl_cipher'").Row().Scan(&variableName, &sslStatus)
		if err != nil {
			s.Close()
			return fmt.Errorf("failed to verify SSL connection: %w", err)
		}
		if sslStatus == "" {
			s.Close()
			// The error is now correctly tied to the configuration requirement
			return fmt.Errorf("CRITICAL: SSL connection required (MYSQL_SSL_MODE=production) but connection is UNENCRYPTED")
		}
		log.Printf("✅ SSL connection established with cipher: %s", sslStatus)
	} else {
		// For any other mode (like 'false'), just log a warning and continue
		log.Printf("⚠️ SSL not required by config. Proceeding with a potentially unencrypted database connection.")
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		s.Close()
		return err
	}

	sqlDB.SetConnMaxLifetime(1 * time.Hour)
	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetMaxOpenConns(512)

	if err = s.db.AutoMigrate(&ShortCodeInfo{}); err != nil {
		log.Printf("%s", fmt.Sprintf("Failed to create/update db tables with error %s", err.Error()))
		os.Exit(1)
	}

	return nil
}

// Close terminates the database connection.
func (s *Store) Close() {
	db, _ := s.db.DB()
	if err := db.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not close database connection: %v\n", err)
		os.Exit(1)
	}
}

// Persist saves a ShortCodeInfo record to the database.
func (s *Store) Persist(info *ShortCodeInfo) error {
	s.db.Save(info)
	return nil
}

// UpsertByShortCode upserts a record by its unique short_code to avoid conflicts on reseed.
func (s *Store) UpsertByShortCode(info *ShortCodeInfo) error {
	info.EndTime = ToDBLocalMicro(info.EndTime)
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "short_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"url", "updated_at", "end_time", "created_by"}),
	}).Create(info).Error
}

// UpsertMany iterates over a slice of ShortCodeInfo and upserts each one.
func (s *Store) UpsertMany(infos []*ShortCodeInfo) error {
	for _, i := range infos {
		if err := s.UpsertByShortCode(i); err != nil {
			return err
		}
	}
	return nil
}

// FindByShortCode retrieves the most recently updated ShortCodeInfo for a given short code.
func (s *Store) FindByShortCode(shortCode string) *ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Order("updated_at desc").Find(&infos, "short_code = ?", shortCode)
	if len(infos) == 0 {
		return nil
	}
	urlInfo := infos[0]
	return &urlInfo
}

// FindByEndTime finds records with an exact match on end_time (normalized to µs).
// NOTE: this uses "=" and not a "BETWEEN" clause.
func (s *Store) FindByEndTime(t time.Time) []ShortCodeInfo {
	t = ToDBLocalMicro(t) // match what we store
	var infos []ShortCodeInfo
	s.db.Where("end_time = ?", t).Order("short_code asc").Find(&infos)
	return infos
}

// Sentinel helpers define special date values for testing or special cases.
var (
	SentinelStart = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	SentinelMax   = time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC) // .999999
)

// FindSentinels retrieves records whose end_time matches one of the sentinel date values.
func (s *Store) FindSentinels() []ShortCodeInfo {
	ss := ToDBLocalMicro(SentinelStart)
	sm := ToDBLocalMicro(SentinelMax)
	var infos []ShortCodeInfo
	s.db.Where("end_time IN (?, ?)", ss, sm).Order("end_time asc, short_code asc").Find(&infos)
	return infos
}

// FindSeededDates finds records created for the demo set (CreatedBy == "keploy.io/dates").
func (s *Store) FindSeededDates() []ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Where("created_by = ?", "keploy.io/dates").Order("end_time asc, short_code asc").Find(&infos)
	return infos
}

// FindActive retrieves all records that have not yet expired.
func (s *Store) FindActive() []ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Where("end_time > ?", time.Now()).Find(&infos)
	return infos
}
