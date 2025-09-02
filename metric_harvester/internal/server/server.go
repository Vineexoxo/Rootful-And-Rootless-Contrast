package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"metric_harvester/internal/collectors"
	"metric_harvester/internal/config"
	"metric_harvester/internal/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server is the main server struct
type Server struct {
	config     *config.Config
	logger     *zap.Logger
	httpServer *http.Server
	registry   *prometheus.Registry
	collectors []collectors.Collector
}

// ServerParams is the parameters for the server
type ServerParams struct {
	Config   *config.Config
	Logger   *zap.Logger
	Executor *utils.SystemCommandExecutor
}

// New creates a new server
// Args:
// - params: ServerParams
// Returns:
// - *Server: new Server instance
func New(params *ServerParams) *Server {
	registry := prometheus.NewRegistry()

	// Create collector dependencies
	deps := &collectors.CollectorDependencies{
		Executor: params.Executor,
		Logger:   params.Logger,
		Config:   params.Config,
	}

	// Initialize collectors
	system_collector := collectors.NewSystemCollector(deps)
	container_collector := collectors.NewContainerCollector(deps)
	network_collector := collectors.NewNetworkCollector(deps)

	// Register collectors with Prometheus
	registry.MustRegister(system_collector)
	registry.MustRegister(container_collector)
	registry.MustRegister(network_collector)
	

	collectors := []collectors.Collector{
		system_collector,
		container_collector,
		network_collector,
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	// Info endpoint
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		info := fmt.Sprintf(`{
			"service": "metric_harvester",
			"collectors": %d,
			"docker_enabled": %t,
			"podman_enabled": %t,
			"collection_interval": "%s"
		}`,
			len(collectors),
			params.Config.Containers.DockerEnabled,
			params.Config.Containers.PodmanEnabled,
			params.Config.Metrics.CollectionInterval,
		)
		w.Write([]byte(info))
	})

	httpServer := &http.Server{
		Addr:         params.Config.Server.Port,
		Handler:      mux,
		ReadTimeout:  params.Config.Server.ReadTimeout,
		WriteTimeout: params.Config.Server.WriteTimeout,
	}

	return &Server{
		config:     params.Config,
		logger:     params.Logger,
		httpServer: httpServer,
		registry:   registry,
		collectors: collectors,
	}
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	// Start metric collection in background
	go s.startMetricCollection(ctx)

	s.logger.Info("Starting HTTP server",
		zap.String("addr", s.httpServer.Addr),
		zap.Duration("read_timeout", s.config.Server.ReadTimeout),
		zap.Duration("write_timeout", s.config.Server.WriteTimeout),
	)

	// Start HTTP server
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("HTTP server failed", zap.Error(err))
		return err
	}

	return nil
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.Server.ShutdownTimeout)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}

// startMetricCollection starts the metric collection
// It collects metrics at the specified interval
func (s *Server) startMetricCollection(ctx context.Context) {
	ticker := time.NewTicker(s.config.Metrics.CollectionInterval)
	defer ticker.Stop()

	s.logger.Info("Starting metric collection",
		zap.Duration("interval", s.config.Metrics.CollectionInterval),
		zap.Int("collectors", len(s.collectors)),
	)

	// Collect metrics immediately on startup
	s.collectAllMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping metric collection")
			return
		case <-ticker.C:
			s.collectAllMetrics(ctx)
		}
	}
}

// collectAllMetrics collects all the metrics
// It collects metrics from all the collectors
// It can be used to collect metrics on demand. For example, when the server is started, the metrics are collected immediately.
// Or when the server is stopped, the metrics are collected immediately.
// It calls the CollectMetrics method of all the collectors.
func (s *Server) collectAllMetrics(ctx context.Context) {
	start := time.Now()

	// Create a timeout context for metric collection
	collectCtx, cancel := context.WithTimeout(ctx, s.config.Metrics.CommandTimeout)
	defer cancel()

	for _, collector := range s.collectors {
		if err := collector.CollectMetrics(collectCtx); err != nil {
			s.logger.Error("Failed to collect metrics",
				zap.String("collector", collector.Name()),
				zap.Error(err),
			)
		}
	}

	duration := time.Since(start)
	s.logger.Debug("Metric collection completed",
		zap.Duration("duration", duration),
		zap.Int("collectors", len(s.collectors)),
	)
}

// ServerLifecycle manages the server lifecycle with fx
type ServerLifecycle struct {
	server *Server
	logger *zap.Logger
}

func NewServerLifecycle(server *Server, logger *zap.Logger) *ServerLifecycle {
	return &ServerLifecycle{
		server: server,
		logger: logger,
	}
}

func (sl *ServerLifecycle) Start(ctx context.Context) error {
	go func() {
		if err := sl.server.Start(ctx); err != nil {
			sl.logger.Error("Server startup failed", zap.Error(err))
		}
	}()
	return nil
}

func (sl *ServerLifecycle) Stop(ctx context.Context) error {
	return sl.server.Stop(ctx)
}
