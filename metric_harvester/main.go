package main

import (
	"fmt"
	"metric_harvester/internal/config"
	"metric_harvester/internal/server"
	"metric_harvester/internal/utils"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

var configPath = "internal/config/configurations.json"

func main() {
	app := fx.New(
		// Provide dependencies
		fx.Provide(
			zap.NewDevelopment,
			// Load configuration from JSON file to config.Config
			func() *config.Config {
				cfg, err := config.LoadFromJSON(configPath)
				if err != nil {
					panic(fmt.Sprintf("Failed to load configuration: %v", err))
				}
				return cfg
			},
			utils.NewSystemCommandExecutor,
			server.New,
			server.NewServerLifecycle,
		),

		// Invoke startup functions
		fx.Invoke(
			func(lifecycle fx.Lifecycle, serverLifecycle *server.ServerLifecycle) {
				lifecycle.Append(fx.Hook{
					OnStart: serverLifecycle.Start,
					OnStop:  serverLifecycle.Stop,
				})
			},
		),

		// Configure logging
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)

	app.Run()
}
