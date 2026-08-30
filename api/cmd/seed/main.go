package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"hesab/api/internal/config"
	"hesab/api/internal/infrastructure/db"
	"log"
	"time"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, e := db.NewPool(ctx, cfg.DatabaseURL)
	if e != nil {
		log.Fatal(e)
	}
	defer pool.Close()
	h, e := bcrypt.GenerateFromPassword([]byte("Amir@Pass1999"), bcrypt.DefaultCost)
	if e != nil {
		log.Fatal(e)
	}
	var id int64
	e = pool.QueryRow(ctx, `INSERT INTO admins (first_name,last_name,email,phone_number,is_male,password_hash,totp_secret) VALUES ($1,$2,$3,$4,$5,$6,'') ON CONFLICT (phone_number) DO UPDATE SET first_name=EXCLUDED.first_name,last_name=EXCLUDED.last_name,email=EXCLUDED.email,password_hash=EXCLUDED.password_hash RETURNING id`, "Amir", "Admin", "admin@hesab.local", "9370843199", true, string(h)).Scan(&id)
	if e != nil && e != pgx.ErrNoRows {
		log.Fatal(e)
	}
	log.Printf("seeded admin id=%d phone=9370843199", id)
	h, e = bcrypt.GenerateFromPassword([]byte("Client@Pass1999"), bcrypt.DefaultCost)
	if e != nil {
		log.Fatal(e)
	}
	e = pool.QueryRow(ctx, `INSERT INTO users (first_name,last_name,email,phone_number,password_hash,totp_secret) VALUES ($1,$2,$3,$4,$5,'') ON CONFLICT (phone_number) DO UPDATE SET first_name=EXCLUDED.first_name,last_name=EXCLUDED.last_name,email=EXCLUDED.email,password_hash=EXCLUDED.password_hash RETURNING id`, "تست", "کاربر", "user@hesab.local", "9120000000", string(h)).Scan(&id)
	if e != nil && e != pgx.ErrNoRows {
		log.Fatal(e)
	}
	log.Printf("seeded user id=%d phone=9120000000", id)
	var bizID int64
	e = pool.QueryRow(ctx, `INSERT INTO businesses (name, owner_user_id)
		SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM businesses WHERE owner_user_id = $2 AND name = $1 AND deleted_at IS NULL)
		RETURNING id`, "کسب‌وکار نمونه", id).Scan(&bizID)
	if e == nil {
		_, e = pool.Exec(ctx, `INSERT INTO business_members (business_id, user_id, role) VALUES ($1, $2, 'owner') ON CONFLICT (business_id, user_id) DO NOTHING`, bizID, id)
	}
	if e != nil && e != pgx.ErrNoRows {
		log.Fatal(e)
	}
	log.Printf("seeded business id=%d owner=%d", bizID, id)
}
