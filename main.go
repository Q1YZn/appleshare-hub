package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/config"
	"github.com/Q1YZn/appleshare-hub/internal/httpapi"
	"github.com/Q1YZn/appleshare-hub/internal/provider"
	"github.com/Q1YZn/appleshare-hub/internal/service"
)

//go:embed web
var webFS embed.FS

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	providers := make([]provider.Provider, 0, len(cfg.Providers))
	for _, pc := range cfg.Providers {
		if !pc.Enabled {
			continue
		}
		p, err := provider.Build(pc)
		if err != nil {
			log.Fatalf("provider %s: %v", pc.ID, err)
		}
		providers = append(providers, p)
	}
	if len(providers) == 0 {
		log.Fatal("no enabled providers, check config.json")
	}

	svc := service.New(providers, cfg.CacheTTL())
	router := httpapi.NewRouter(svc, webFS)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("AppleShare Hub listening on http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func loadConfig() (config.Config, error) {
	cfg, err := config.Load("config.json")
	if err == nil {
		return cfg, nil
	}
	if !os.IsNotExist(err) {
		return config.Config{}, err
	}
	return config.Default(), nil
}
