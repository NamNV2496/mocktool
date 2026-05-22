package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/namnv2496/mocktool/cmd/mcpserver"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/internal/tools"
)

var mcpServerCmd = &cobra.Command{
	Use:   "mcpserver",
	Short: "Start the MCP-over-SSE adapter for mocktool tools",
	RunE: func(cmd *cobra.Command, _ []string) error {
		app := buildMCPServerApp()
		return app.Start(cmd.Context())
	},
}

func buildMCPServerApp() *fx.App {
	config := configs.LoadConfig()
	return fx.New(
		fx.StartTimeout(15*time.Second),
		fx.StopTimeout(30*time.Second),
		fx.Supply(config),
		fx.Provide(
			repository.NewMongoConnect,
			fx.Annotate(repository.NewFeatureRepository, fx.As(new(repository.IFeatureRepository))),
			fx.Annotate(repository.NewScenarioRepository, fx.As(new(repository.IScenarioRepository))),
			fx.Annotate(repository.NewAccountScenarioRepository, fx.As(new(repository.IAccountScenarioRepository))),
			fx.Annotate(repository.NewMockAPIRepository, fx.As(new(repository.IMockAPIRepository))),
			fx.Annotate(repository.NewCache, fx.As(new(repository.ICache))),
			buildToolsDeps,
		),
		mcpserver.Module(),
	)
}

// buildToolsDeps assembles tools.Deps from the fx-injected repos. Used by both
// mcpserver and slackbot subcommands.
func buildToolsDeps(
	feature repository.IFeatureRepository,
	scenario repository.IScenarioRepository,
	account repository.IAccountScenarioRepository,
	api repository.IMockAPIRepository,
	cache repository.ICache,
) tools.Deps {
	return tools.Deps{
		Feature:         feature,
		Scenario:        scenario,
		AccountScenario: account,
		MockAPI:         api,
		Cache:           cache,
	}
}
