package cmd

import (
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
		fx.StartTimeout(time.Second*10),
		fx.StopTimeout(time.Second*10),
		fx.Provide(
			fx.Annotate(repository.NewFeatureRepository, fx.As(new(repository.IFeatureRepository))),
			fx.Annotate(repository.NewScenarioRepository, fx.As(new(repository.IScenarioRepository))),
			fx.Annotate(repository.NewAccountScenarioRepository, fx.As(new(repository.IAccountScenarioRepository))),
			fx.Annotate(repository.NewMockAPIRepository, fx.As(new(repository.IMockAPIRepository))),

			fx.Annotate(controller.NewMockController, fx.As(new(controller.IMockController))),
			// fx.Annotate(controller.NewFowardController, fx.As(new(controller.IFowardController))),
			fx.Annotate(usecase.NewTrie, fx.As(new(usecase.ITrie))),
			fx.Annotate(usecase.NewForwardUC, fx.As(new(usecase.IForwardUC))),

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
	mockController controller.IMockController,
) error {
	return mockController.StartHttpServer()
}
