package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Init() error {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "app")
	password := getEnv("DB_PASSWORD", "app")
	dbname := getEnv("DB_NAME", "bookdb")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		user, password, host, port, dbname)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	DB.SetMaxOpenConns(100)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(time.Hour)

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		err = DB.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
