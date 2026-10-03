package main

import (
	"chat_app/internal/db"
	"chat_app/internal/hub"
	"chat_app/internal/routes"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// httpShutdownTimeout bounds how long in-flight HTTP requests are given to
// finish after a shutdown signal.
const httpShutdownTimeout = 10 * time.Second

// readHeaderTimeout bounds how long a client may take to send its request
// headers, including a WebSocket handshake.
const readHeaderTimeout = 10 * time.Second

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	queries, pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	wsConfig, err := hub.ConfigFromEnv()
	if err != nil {
		log.Fatalf("invalid websocket configuration: %v", err)
	}
	realtimeHub := hub.New(wsConfig)

	r := routes.MakeRouter(queries, realtimeHub)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-shutdownSignal.Done()

		httpCtx, cancelHTTP := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancelHTTP()
		if err := srv.Shutdown(httpCtx); err != nil {
			log.Printf("http shutdown failed: %v", err)
		}

		// WebSocket connections are hijacked, so Shutdown above never sees
		// them and would report success while clients are still attached.
		// Closing them explicitly is what lets in-flight socket writes finish
		// before the process exits.
		realtimeCtx, cancelRealtime := context.WithTimeout(context.Background(), wsConfig.ShutdownTimeout)
		defer cancelRealtime()
		realtimeHub.Shutdown(realtimeCtx)
	}()

	log.Printf("server starting on port %s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}

	<-shutdownDone
	log.Print("server stopped")
}
