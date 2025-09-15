// Filename: cmd/api/main.go

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type configuration struct {
	port int
	env  string
	vrs  string
	db   struct {
		dsn string
	}
}
type application struct {
	config configuration
	logger *slog.Logger
}

func printUB() string {
	return "Hello, UB!"
}

func main() {
	greeting := printUB()
	fmt.Println(greeting)

	//Initialize configuration
	cfg := loadConfig()
	//Initialize logger
	logger := setupLogger()

	// the call to openDB() sets up our connection pool
	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	// release the database resources before exiting
	defer db.Close()

	logger.Info("database connection pool established")

	//Initialize applicatioin with dependencies
	app := &application{
		config: cfg,
		logger: logger,
	}

	// Start the application server
	if err := app.serve(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

}

func loadConfig() configuration {
	var cfg configuration

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")
	flag.StringVar(&cfg.vrs, "version", "1.0.0", "Application version")
	// read in the dsn
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://quotes:root@localhost/quotes?sslmode=disable", "PostgreSQL DSN")
	flag.Parse()

	return cfg
}

func setupLogger() *slog.Logger {
	var logger *slog.Logger

	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	return logger
}

func openDB(settings configuration) (*sql.DB, error) {
	// open a connection pool
	db, err := sql.Open("postgres", settings.db.dsn)
	if err != nil {
		return nil, err
	}

	// set a context to ensure DB operations don't take too long
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// let's test if the connection pool was created
	// we trying pinging it with a 5-second timeout
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	// return the connection pool (sql.DB)
	return db, nil

}
