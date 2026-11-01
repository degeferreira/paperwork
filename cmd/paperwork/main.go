// Command paperwork runs the standalone local Paperwork API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	paperwork "github.com/degeferreira/paperwork/go"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func run() error {
	store, err := paperwork.Open(env("PAPERWORK_HOME", "/paperwork"))
	if err != nil {
		return err
	}
	defer store.Close()
	handler, err := paperwork.NewServer(store)
	if err != nil {
		return err
	}
	defer handler.Close()
	handler.StartIntegrationWorkers()
	defer handler.Client.CloseIdleConnections()
	server := &http.Server{
		Addr: env("PAPERWORK_LISTEN", "127.0.0.1:8089"), Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second,
		WriteTimeout: 180 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	select {
	case err = <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = server.Shutdown(shutdown)
		if err != nil {
			_ = server.Close()
		}
		return err
	}
}
func main() {
	if err := run(); err != nil {
		// Errors may contain local paths/configuration; avoid logging their contents.
		log.Printf("Paperwork stopped (%T)", err)
		os.Exit(1)
	}
}
