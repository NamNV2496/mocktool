package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/namnv2496/mocktool/cmd/slackbot"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/repository"
)

var slackBotCmd = &cobra.Command{
	Use:   "slackbot",
	Short: "Start the Slack Socket Mode bot that uses OpenAI to drive mocktool tools",
	RunE: func(cmd *cobra.Command, _ []string) error {
		app := buildSlackBotApp()
		return app.Start(cmd.Context())
	},
}

func buildSlackBotApp() *fx.App {
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
		slackbot.Module(),
	)
}
