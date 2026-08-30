package main

import (
	"context"
	"log"
	"time"

	"hesab/api/internal/application/adminauth"
	"hesab/api/internal/application/health"
	"hesab/api/internal/config"
	"hesab/api/internal/infrastructure/db"
	"hesab/api/internal/infrastructure/db/sqlc"
	"hesab/api/internal/infrastructure/repo"
	"hesab/api/internal/infrastructure/sms"
	"hesab/api/internal/infrastructure/token"
	httpiface "hesab/api/internal/interface/http"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	tokens := token.New(cfg)
	authSvc := adminauth.NewService(repo.NewAdminRepo(sqlc.New(pool)), tokens, sms.FakeSender{Log: log.Default()}, func() string { return sms.FixedCode }, time.Now, cfg) // TODO: real generator = crypto/rand 6-digit
	router := httpiface.NewRouter(health.NewService(pool), authSvc, tokens, cfg)

	log.Printf("listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
