// Filename: cmd/api/main.go

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"sync"

	"github.com/2016114132/qod/internal/mailer"

	"github.com/2016114132/qod/internal/data"
	_ "github.com/lib/pq"
)

type configuration struct {
	port int
	env  string
	vrs  string
	db   struct {
		dsn string
	}
	cors struct {
		trustedOrigins []string
	}
	limiter struct {
		rps     float64 // requests per second
		burst   int     // initial requests possible
		enabled bool    // enable or disable rate limiter
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}
type application struct {
	config     configuration
	logger     *slog.Logger
	quoteModel data.QuoteModel
	userModel  data.UserModel
	mailer     mailer.Mailer
	wg         sync.WaitGroup // need this later for background jobs
	tokenModel data.TokenModel
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
		config:     cfg,
		logger:     logger,
		quoteModel: data.QuoteModel{DB: db},
		userModel:  data.UserModel{DB: db},
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port,
			cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		tokenModel: data.TokenModel{DB: db},
	}

	// Start the application server
	err = app.serve()
	if err != nil {
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

	// We will build a custom command-line flag.  This flag will allow us to access
	// space-separated origins. We will then put those origins in our slice. Again not // something we can do with the flag functions that we have seen so far.
	// strings.Fields() splits string (origins) on spaces
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)",
		func(val string) error {
			cfg.cors.trustedOrigins = strings.Fields(val)
			return nil
		})

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2,
		"Rate Limiter maximum requests per second")

	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 5,
		"Rate Limiter maximum burst")

	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true,
		"Enable rate limiter")

	flag.StringVar(&cfg.smtp.host,
		"smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	// We have port 25, 465, 587, 2525. If 25 doesn't work choose another
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")

	// Use your Username value provided by Mailtrap
	flag.StringVar(&cfg.smtp.username, "smtp-username",
		"c5a15ec98269db", "SMTP username")

	// Use your Password value provided by Mailtrap
	flag.StringVar(&cfg.smtp.password, "smtp-password",
		"424567a8b96fde", "SMTP password")

	flag.StringVar(&cfg.smtp.sender, "smtp-sender",
		"QOD <no-reply@ub.edu.bz>",
		"SMTP sender")

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
