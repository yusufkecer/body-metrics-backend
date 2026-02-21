package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yusufkecer/body-metrics-backend/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	const maxRetries = 10
	const retryInterval = 5 * time.Second

	for i := 1; i <= maxRetries; i++ {
		if err = db.Ping(); err == nil {
			log.Println("database connection established")
			return db, nil
		}
		log.Printf("database not ready (attempt %d/%d): %v", i, maxRetries, err)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}
