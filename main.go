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

	app_reg "onibi/applications_registry"
	"onibi/server"
	user_reg "onibi/users_registry"
)

// add sqlc
// add jwt

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	go func() { <-ctx.Done(); stop() }()
	addr := os.Getenv("ADDRESS")
	JWTKey := os.Getenv("JWTKEY")
	api := server.API{
		Applications: app_reg.NewInMemoryApplicationStore(),
		Users:        user_reg.NewUsersRegistry([]byte(JWTKey)),
		MaxReadBytes: 10 * 1024 * 1024}
	srv := &http.Server{
		Addr:         addr,
		Handler:      api.GetMux(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failes to start: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
