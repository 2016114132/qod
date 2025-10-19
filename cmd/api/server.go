package main

import (
	"context"
	"errors"

	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	// create a channel to keep track of any errors during the shutdown process
	shutdownError := make(chan error)
	// create a goroutine that runs in the background listening
	// for the shutdown signals
	go func() {
		quit := make(chan os.Signal, 1)                      // receive the shutdown signal
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // signal occurred
		s := <-quit                                          // blocks until a signal is received
		// message about shutdown in process
		app.logger.Info("shutting down server", "signal", s.String())
		// create a context
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// we will only write to the error channel if there is an error
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		// Wait for background tasks to complete
		app.logger.Info("completing background tasks", "address", srv.Addr)
		app.wg.Wait()
		shutdownError <- nil // successful shutdown

		// initiate the shutdown. If all okay this returns nil
		// shutdownError <- srv.Shutdown(ctx)
	}()

	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.env)
	// return srv.ListenAndServe()
	// something went wrong during shutdown if we don't get ErrServerClosed()
	// this only happens when we issue the shutdown command from our goroutine
	// otherwise our server keeps running as normal as it should.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// check the error channel to see if there were shutdown errors
	err = <-shutdownError
	if err != nil {
		return err
	}
	// graceful shutdown was successful
	app.logger.Info("stopped server", "address", srv.Addr)

	return nil

}
