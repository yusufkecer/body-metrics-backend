package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type migration struct {
	version string
	sql     string
}

var migrations = []migration{
	{
		version: "000_create_accounts",
		sql: `
			CREATE TABLE IF NOT EXISTS accounts (
				id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
				email         VARCHAR(255) NOT NULL UNIQUE,
				password_hash VARCHAR(255) NOT NULL,
				created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`,
	},
	{
		version: "001_create_users",
		sql: `
			CREATE TABLE IF NOT EXISTS users (
				id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
				name          VARCHAR(100),
				surname       VARCHAR(100),
				gender        TINYINT,
				avatar        VARCHAR(50),
				height        INT,
				birth_of_date VARCHAR(20),
				created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`,
	},
	{
		version: "002_create_user_metrics",
		sql: `
			CREATE TABLE IF NOT EXISTS user_metrics (
				id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
				user_id     BIGINT UNSIGNED NOT NULL,
				date        VARCHAR(20) NOT NULL,
				weight      DOUBLE,
				height      INT NOT NULL,
				bmi         DOUBLE NOT NULL,
				weight_diff DOUBLE,
				body_metric VARCHAR(30),
				created_at  VARCHAR(30),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			)`,
	},
}

func RunMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(255) PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := executeMigration(db, m); err != nil {
			return err
		}

		log.Printf("applied migration: %s", m.version)
	}

	return nil
}

func isMigrationApplied(db *sql.DB, version string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
		version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check migration %s: %w", version, err)
	}
	return count > 0, nil
}

func executeMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for %s: %w", m.version, err)
	}

	for _, stmt := range strings.Split(m.sql, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", m.version, err)
		}
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES (?)",
		m.version,
	); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration %s: %w", m.version, err)
	}

	return tx.Commit()
}
