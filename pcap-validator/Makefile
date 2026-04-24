.PHONY: compose-up deps run traffic-plain traffic-tls down

compose-up:
	docker compose up --build

deps:
	docker compose up -d postgres mongo

run:
	DATABASE_URL='postgres://postgres:postgres@localhost:5432/pcapdemo?sslmode=disable' \
	MONGO_URI='mongodb://localhost:27017' \
	go run .

traffic-plain:
	curl -s http://localhost:8080/healthz
	curl -s -X POST http://localhost:8080/touch \
		-H 'content-type: application/json' \
		-d '{"label":"plain-http-touch"}'

traffic-tls:
	curl -sk https://localhost:8443/healthz
	curl -sk -X POST https://localhost:8443/touch \
		-H 'content-type: application/json' \
		-d '{"label":"tls-https-touch"}'

down:
	docker compose down -v
