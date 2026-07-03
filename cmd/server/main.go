package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/yuya-cpu/web-security-proxy/internal/config"
	"github.com/yuya-cpu/web-security-proxy/internal/fetch"
	"github.com/yuya-cpu/web-security-proxy/internal/handler"
	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/proxy"
	"github.com/yuya-cpu/web-security-proxy/internal/repository"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	configPath := envOrDefault("CONFIG_PATH", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	migrationSQL, err := os.ReadFile("db/migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}

	repo, err := repository.NewSQLiteTrafficRepository(cfg.Database.Path, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("init repository: %w", err)
	}
	defer repo.Close()

	trafficService := service.NewTrafficService(repo)
	repeaterService := service.NewRepeaterService(repo, nil)
	diagnosticService := service.NewDiagnosticService(scanner.NewDiagnosticScanner())
	scanService := service.NewScanService(scanner.NewActiveScanner(fetch.NewHTTPFetcher(nil)))
	recorder := &serviceRecorder{svc: trafficService}

	templates, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	webHandler := handler.NewHandler(trafficService, repeaterService, diagnosticService, scanService, templates)
	webMux := http.NewServeMux()
	webHandler.RegisterRoutes(webMux)

	proxyServer := &http.Server{
		Addr:              cfg.Proxy.Addr(),
		Handler:           proxy.NewServer(recorder),
		ReadHeaderTimeout: 10 * time.Second,
	}

	webServer := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           webMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	startServer := func(name string, server *http.Server) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("%s listening on %s", name, server.Addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("%s: %w", name, err)
			}
		}()
	}

	startServer("proxy", proxyServer)
	startServer("web", webServer)

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := proxyServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("proxy shutdown error: %v", err)
	}
	if err := webServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("web shutdown error: %v", err)
	}

	wg.Wait()
	log.Println("shutdown complete")
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type serviceRecorder struct {
	svc *service.TrafficService
}

func (r *serviceRecorder) SaveTransaction(ctx context.Context, tx *model.HTTPTransaction) (int64, error) {
	return r.svc.SaveTransaction(ctx, tx)
}
