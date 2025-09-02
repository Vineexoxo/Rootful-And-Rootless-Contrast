package collectors

import (
	"context"
	"metric_harvester/internal/config"
	"metric_harvester/internal/utils"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Collector interface {
	prometheus.Collector
	Name() string
	CollectMetrics(ctx context.Context) error
}

type CollectorDependencies struct {
	Executor *utils.SystemCommandExecutor
	Logger   *zap.Logger
	Config   *config.Config
}
