DATABASE_URL ?= postgres://hesab:hesab@localhost:5432/hesab?sslmode=disable
migrate-up: ; migrate -path db/migration -database "$(DATABASE_URL)" up
migrate-down: ; migrate -path db/migration -database "$(DATABASE_URL)" down 1
migrate-create: ; migrate create -ext sql -dir db/migration -seq $(name)
sqlc: ; sqlc generate
seed: ; go run ./cmd/seed
run: ; go run ./cmd/server
test: ; go test ./...
