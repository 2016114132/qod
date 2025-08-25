// Filename: cmd/api/main.go

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

type configuration struct {
	port int
	env  string
	vrs  string
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
	flag.Parse()

	return cfg
}

func setupLogger() *slog.Logger {
	var logger *slog.Logger

	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	return logger
}
