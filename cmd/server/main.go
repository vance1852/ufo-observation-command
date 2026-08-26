package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"ufo-observation-command/internal/config"
	"ufo-observation-command/internal/console"
	"ufo-observation-command/internal/db"
	"ufo-observation-command/internal/httpapi"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
	"ufo-observation-command/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	p, err := db.Open(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	if err != nil {
		return err
	}
	defer p.Close()
	if err := db.Migrate(ctx, p); err != nil {
		return err
	}

	repo := repository.NewPostgres(p)
	svc := service.New(repo)
	api := httpapi.New(svc, p.Ping).WithConsole(console.NewStore(p))
	server := &http.Server{Addr: cfg.Addr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	workers := []struct {
		name string
		run  func(context.Context) error
	}{
		{name: "assignment", run: worker.NewAssignmentWorker(repo, cfg.WorkerInterval, logger, nil).Run},
		{name: "task-expiration", run: worker.NewPeriodic(cfg.WorkerInterval, worker.NewExpirationReconciler(svc, logger, nil), logger).Run},
		{name: "safety_alert", run: worker.NewIntegrityIncidentWorker(svc, cfg.WorkerInterval, logger).Run},
	}
	workerErr := make(chan error, len(workers))
	workersDone := make(chan struct{})
	var workerWG sync.WaitGroup
	for _, item := range workers {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			if err := item.run(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
				workerErr <- fmt.Errorf("%s worker: %w", item.name, err)
			}
		}()
	}
	go func() {
		workerWG.Wait()
		close(workersDone)
	}()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.Addr)
		serverErr <- server.ListenAndServe()
	}()
	var runErr error
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = fmt.Errorf("http server: %w", err)
		}
	case err := <-workerErr:
		runErr = err
	case <-ctx.Done():
	}
	cancelWorkers()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown http server: %w", err)
	}
	select {
	case <-workersDone:
	case <-time.After(cfg.ShutdownTimeout):
		return fmt.Errorf("worker shutdown timeout")
	}
	return runErr
}
