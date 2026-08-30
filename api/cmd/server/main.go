package main

import (
	"context"
	"log"
	"time"

	"hesab/api/internal/application/adminauth"
	"hesab/api/internal/application/clientauth"
	"hesab/api/internal/application/health"
	"hesab/api/internal/application/usersadmin"
	"hesab/api/internal/config"
	"hesab/api/internal/infrastructure/db"
	"hesab/api/internal/infrastructure/db/sqlc"
	"hesab/api/internal/infrastructure/repo"
	"hesab/api/internal/infrastructure/sms"
	"hesab/api/internal/infrastructure/token"
	httpiface "hesab/api/internal/interface/http"
)

// clientTokenAdapter keeps client token types separate from admin tokens.
type clientTokenAdapter struct{ token.JWT }

func (a clientTokenAdapter) IssueAccess(id int64) (string, int, error) {
	return a.IssueClientAccess(id)
}
func (a clientTokenAdapter) IssuePending(id int64) (string, error) { return a.IssueClientPending(id) }
func (a clientTokenAdapter) ParseAccess(s string) (int64, error)   { return a.ParseClientAccess(s) }
func (a clientTokenAdapter) ParsePending(s string) (int64, error)  { return a.ParseClientPending(s) }

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
	clientTokens := clientTokenAdapter{tokens}
	clientSvc := clientauth.NewService(repo.NewUserRepo(sqlc.New(pool)), clientTokens, sms.FakeSender{Log: log.Default()}, func() string { return sms.FixedCode }, time.Now, cfg)
	usersAdminSvc := usersadmin.NewService(repo.NewUserAdminRepo(sqlc.New(pool)), sms.FakeSender{Log: log.Default()})
	router := httpiface.NewRouter(health.NewService(pool), authSvc, tokens, usersAdminSvc, clientSvc, clientTokens, cfg)

	log.Printf("listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
