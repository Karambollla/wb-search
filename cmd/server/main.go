package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	searchv1 "github.com/Karambollla/wb-search/api/proto/searchv1"
	grpcadapter "github.com/Karambollla/wb-search/internal/adapters/grpc"
	httpadapter "github.com/Karambollla/wb-search/internal/adapters/http"
	metricsadapter "github.com/Karambollla/wb-search/internal/adapters/metrics"
	natsadapter "github.com/Karambollla/wb-search/internal/adapters/nats"
	"github.com/Karambollla/wb-search/internal/config"
	"github.com/Karambollla/wb-search/internal/core"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	coreService := core.NewService(core.RealClock{}, core.ServiceConfig{
		Window:          cfg.Window,
		BucketDuration:  cfg.BucketDuration,
		FraudTTL:        cfg.FraudTTL,
		MaxTopSize:      cfg.MaxTopSize,
		InitialStopList: cfg.StopWords,
	})

	registry := prometheus.NewRegistry()
	metrics := metricsadapter.New(registry)
	service := metricsadapter.NewInstrumentedService(coreService, metrics)

	go coreService.StartTopRefresher(ctx, cfg.RefreshEvery)

	subscriber, err := natsadapter.NewSubscriber(cfg.NATSURL, cfg.NATSSubject, service, log)
	if err != nil {
		return err
	}
	defer func() {
		if err := subscriber.Close(); err != nil {
			log.Warn("failed to close nats subscriber", "error", err)
		}
	}()
	go subscriber.Start(ctx)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpadapter.NewServer(service, promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	grpcServer := grpcserver.NewServer()
	grpcAPI := grpcadapter.NewServer(service)
	searchv1.RegisterTopServiceServer(grpcServer, grpcAPI)
	searchv1.RegisterStopListServiceServer(grpcServer, grpcAPI)
	reflection.Register(grpcServer)

	errCh := make(chan error, 2)
	go func() {
		log.Info("starting http server", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		log.Info("starting grpc server", "addr", cfg.GRPCAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}
	grpcServer.GracefulStop()
	return nil
}
