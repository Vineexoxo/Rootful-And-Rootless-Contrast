package main

import (
	"context"
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
			// Provide logger
			zap.NewDevelopment,
			// Load configuration from JSON file to config.Config
			func() *config.Config {
				cfg, err := config.LoadFromJSON(configPath)
				if err != nil {
					panic(fmt.Sprintf("Failed to load configuration: %v", err))
				}
				return cfg
			},
			// Provide system command executor using logger
			utils.NewSystemCommandExecutor,
			// Provide ServerParams using config, logger and executor
			func(cfg *config.Config, logger *zap.Logger, executor *utils.SystemCommandExecutor) *server.ServerParams {
				return &server.ServerParams{
					Config:   cfg,
					Logger:   logger,
					Executor: executor,
				}
			},
			server.New,
		),

		// Invoke startup functions
		fx.Invoke(
			func(lifecycle fx.Lifecycle, server *server.Server) {
				lifecycle.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						go func() {
							if err := server.Start(ctx); err != nil {
								// Server will log the error internally
							}
						}()
						return nil
					},
					OnStop: server.Stop,
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
