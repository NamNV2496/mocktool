// Package slackbot wires Slack Socket Mode to the tools registry via an OpenAI
// function-calling dispatcher. Users mention the bot in a channel (or DM it),
// the dispatcher resolves intent via OpenAI, calls mocktool tools, and replies
// in the same thread.
package slackbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"go.uber.org/fx"

	"github.com/namnv2496/mocktool/internal/tools"
)

// Config is the slackbot configuration sourced from env vars.
type Config struct {
	SlackAppToken    string        `env:"SLACK_APP_TOKEN"`
	SlackBotToken    string        `env:"SLACK_BOT_TOKEN"`
	OpenAIAPIKey     string        `env:"OPENAI_API_KEY"`
	OpenAIBaseURL    string        `env:"OPENAI_BASE_URL"`
	OpenAIModel      string        `env:"OPENAI_MODEL" envDefault:"gpt-4o"`
	MaxIter          int           `env:"LLM_MAX_ITERATIONS" envDefault:"5"`
	ConfirmTTL       time.Duration `env:"CONFIRM_TTL" envDefault:"60s"`
	ThreadMemoryTTL  time.Duration `env:"THREAD_MEMORY_TTL" envDefault:"30m"`
}

// Bot bundles the Slack Socket Mode client and the dispatcher.
type Bot struct {
	cfg        Config
	api        *slack.Client
	smc        *socketmode.Client
	dispatcher *Dispatcher
	botUserID  string
}

// New constructs a bot from explicit dependencies. The tests reuse Dispatcher
// directly and do not call New.
func New(cfg Config, reg *tools.Registry) (*Bot, error) {
	if cfg.SlackAppToken == "" || !strings.HasPrefix(cfg.SlackAppToken, "xapp-") {
		return nil, fmt.Errorf("SLACK_APP_TOKEN must be set to an xapp-... token")
	}
	if cfg.SlackBotToken == "" || !strings.HasPrefix(cfg.SlackBotToken, "xoxb-") {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN must be set to an xoxb-... token")
	}
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY must be set")
	}

	api := slack.New(
		cfg.SlackBotToken,
		slack.OptionAppLevelToken(cfg.SlackAppToken),
	)
	smc := socketmode.New(api)

	llm := NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL)
	mem := newThreadMemory(cfg.ThreadMemoryTTL)
	conf := newConfirmStore(cfg.ConfirmTTL)
	disp := NewDispatcher(llm, cfg.OpenAIModel, reg, mem, conf, cfg.MaxIter)

	return &Bot{cfg: cfg, api: api, smc: smc, dispatcher: disp}, nil
}

// Run starts the Socket Mode event loop. Blocks until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	// Resolve our own user ID so we can strip leading @mentions cleanly.
	auth, err := b.api.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("slack auth.test: %w", err)
	}
	b.botUserID = auth.UserID
	slog.Info("slackbot connected", "bot_user_id", b.botUserID, "team", auth.Team)

	go b.handleEvents(ctx)
	return b.smc.RunContext(ctx)
}

func (b *Bot) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-b.smc.Events:
			if !ok {
				return
			}
			b.routeEvent(ctx, evt)
		}
	}
}

func (b *Bot) routeEvent(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeConnecting:
		slog.Info("slackbot: connecting to Slack")
	case socketmode.EventTypeConnected:
		slog.Info("slackbot: connected")
	case socketmode.EventTypeDisconnect:
		slog.Warn("slackbot: disconnected")
	case socketmode.EventTypeEventsAPI:
		ev, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return
		}
		b.smc.Ack(*evt.Request)
		b.handleEventsAPI(ctx, ev)
	}
}

func (b *Bot) handleEventsAPI(ctx context.Context, ev slackevents.EventsAPIEvent) {
	if ev.Type != slackevents.CallbackEvent {
		return
	}
	switch e := ev.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		b.respond(ctx, e.Channel, threadOf(e.ThreadTimeStamp, e.TimeStamp), b.stripMention(e.Text))
	case *slackevents.MessageEvent:
		// Only respond to direct messages and threaded replies to the bot.
		// Skip the bot's own messages and edits.
		if e.BotID != "" || e.User == b.botUserID || e.SubType != "" {
			return
		}
		if e.ChannelType == "im" || e.ThreadTimeStamp != "" {
			b.respond(ctx, e.Channel, threadOf(e.ThreadTimeStamp, e.TimeStamp), e.Text)
		}
	}
}

func (b *Bot) respond(ctx context.Context, channel, threadTS, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	reply := func(ctx context.Context, msg string) error {
		_, _, err := b.api.PostMessageContext(
			ctx, channel,
			slack.MsgOptionText(msg, false),
			slack.MsgOptionTS(threadTS),
		)
		return err
	}
	if err := b.dispatcher.HandleText(ctx, threadTS, channel, text, reply); err != nil {
		slog.Error("slackbot: handle failed", "error", err, "thread", threadTS)
		_ = reply(ctx, fmt.Sprintf(":x: Something went wrong: `%v`", err))
	}
}

// stripMention removes a leading <@UXXXX> mention of our own bot, leaving the
// real text for the LLM.
func (b *Bot) stripMention(text string) string {
	prefix := fmt.Sprintf("<@%s>", b.botUserID)
	out := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), prefix))
	return out
}

// threadOf returns the thread root if this is a reply, otherwise the message ts
// (so the bot creates a new thread anchored on the user's message).
func threadOf(threadTS, ts string) string {
	if threadTS != "" {
		return threadTS
	}
	return ts
}

// Module is the fx wiring for the slackbot.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(loadConfig),
		fx.Provide(tools.BuildAll),
		fx.Invoke(register),
	)
}

func loadConfig() Config {
	var c Config
	if err := env.Parse(&c); err != nil {
		panic(err)
	}
	return c
}

func register(lc fx.Lifecycle, reg *tools.Registry, cfg Config) error {
	bot, err := New(cfg, reg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := bot.Run(ctx); err != nil && err != context.Canceled {
					slog.Error("slackbot exited", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return nil
}
