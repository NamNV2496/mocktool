package cmd

import (
	"context"
	"log/slog"
	"time"

	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/controller"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/internal/usecase"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var serviceFlags = &entity.ServiceFlags{}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Start the mock tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := InvokeServer(startServer)
		return app.Start(cmd.Context())
	},
}

func init() {
	serviceCmd.Flags().IntVarP(
		&serviceFlags.TestWay,
		"test",
		"t", // short flag
		1,
		"HTTP server port",
	)

	serviceCmd.Flags().BoolVar(
		&serviceFlags.EnableHTTP,
		"http",
		true,
		"Enable HTTP server",
	)
}

func InvokeServer(invokers ...any) *fx.App {
	config := configs.LoadConfig()
	app := fx.New(
		fx.StartTimeout(time.Second*15),
		fx.StopTimeout(time.Second*30),
		fx.Provide(
			fx.Annotate(repository.NewFeatureRepository, fx.As(new(repository.IFeatureRepository))),
			fx.Annotate(repository.NewScenarioRepository, fx.As(new(repository.IScenarioRepository))),
			fx.Annotate(repository.NewAccountScenarioRepository, fx.As(new(repository.IAccountScenarioRepository))),
			fx.Annotate(repository.NewMockAPIRepository, fx.As(new(repository.IMockAPIRepository))),

			fx.Annotate(controller.NewMockController, fx.As(new(controller.IMockController))),
			fx.Annotate(controller.NewFowardController, fx.As(new(controller.IForwardController))),
			fx.Annotate(usecase.NewForwardUC, fx.As(new(usecase.IForwardUC))),
			// load test
			fx.Annotate(controller.NewLoadTestController, fx.As(new(controller.ILoadTestController))),
			fx.Annotate(repository.NewLoadTestScenarioRepository, fx.As(new(repository.ILoadTestScenarioRepository))),

			repository.NewMongoConnect,
		),
		fx.Supply(
			config,
			*serviceFlags,
		),
		fx.Invoke(invokers...),
	)
	return app
}

func startServer(
	lc fx.Lifecycle,
	forwardController controller.IForwardController,
	mockController controller.IMockController,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			slog.Info("Starting mocktool servers...")

			// Start forward controller in background
			go func() {
				if err := forwardController.StartMockServer(); err != nil {
					slog.Error("Forward server error", "error", err)
				}
			}()

			// Start mock controller in background
			go func() {
				if err := mockController.StartHttpServer(); err != nil {
					slog.Error("Mock server error", "error", err)
				}
			}()

			slog.Info("Mocktool servers started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			slog.Info("Shutting down mocktool servers gracefully...")
			// Give servers time to finish processing requests
			time.Sleep(2 * time.Second)
			slog.Info("Mocktool servers stopped")
			return nil
		},
	})
}
