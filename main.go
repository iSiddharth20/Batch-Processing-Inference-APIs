package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iSiddharth20/Batch-Processing-Inference-APIs/internal/config"
	"github.com/iSiddharth20/Batch-Processing-Inference-APIs/internal/utils/response"
)

func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteJson(w, http.StatusOK, "ok")
	}
}

func main() {
	// setup: context object
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// setup: load config
	cfg := config.MustLoad()

	// setup: router
	router := http.NewServeMux()

	// Define endpoints
	router.HandleFunc("GET /health", Health())

	// setup: setup
	server := http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	// channel to accept os interrupts and gracefully shutdown server
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Run async server
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("failed to start server: %s", err.Error())
		}
	}()
	slog.Info("server started", slog.String("address", cfg.Addr))

	// gracefully shutdown server
	<-done
	slog.Info("server shut down initiated")
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed: ", slog.String("error", err.Error()))
	}
	slog.Info("server shutting down successful")
}
