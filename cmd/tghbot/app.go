package main

import (
	"context"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"

	"github.com/tdakkota/tghbot/tghbot"
)

type App struct {
	logger *zap.Logger
	bot    *tghbot.Bot
}

func NewApp() *App {
	logger, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel))
	return &App{
		logger: logger,
	}
}

type ClientCallback = func(c *cli.Context, client *telegram.Client) error

func (app *App) createTelegram(c *cli.Context, dispatcher tg.UpdateDispatcher, cb ClientCallback) error {
	logger := app.logger

	sessionDir := ""
	if c.IsSet("tg.session_dir") {
		sessionDir = c.String("tg.session_dir")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			sessionDir = "./.td"
		} else {
			sessionDir = filepath.Join(home, ".td")
		}
	}
	if err := os.MkdirAll(sessionDir, 0600); err != nil {
		return xerrors.Errorf("failed to create session dir: %w", err)
	}

	client := telegram.NewClient(c.Int("tg.app_id"), c.String("tg.app_hash"), telegram.Options{
		Logger: logger,
		SessionStorage: &session.FileStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
		UpdateHandler: dispatcher.Handle,
	})

	return client.Run(c.Context, func(ctx context.Context) error {
		auth, err := client.AuthStatus(c.Context)
		if err != nil {
			return xerrors.Errorf("failed to get auth status: %w", err)
		}

		logger.With(zap.Bool("authorized", auth.Authorized)).Info("Auth status")
		if !auth.Authorized {
			if err := client.AuthBot(c.Context, c.String("tg.bot_token")); err != nil {
				return xerrors.Errorf("failed to perform bot login: %w", err)
			}
			logger.Info("Bot login ok")
		}

		return cb(c, client)
	})
}

func (app *App) run(c *cli.Context) (err error) {
	dispatcher := tg.NewUpdateDispatcher()
	return app.createTelegram(c, dispatcher, func(c *cli.Context, client *telegram.Client) error {
		options := tghbot.Options{
			PollTimeout: c.Duration("bot.poll_timeout"),
			Template:    nil,
		}
		if c.IsSet("bot.template_path") {
			p, err := filepath.Abs(c.Path("bot.template_path"))
			if err != nil {
				return err
			}
			p = filepath.Join(p, "*")
			options.Template, err = template.ParseGlob(p)
			if err != nil {
				return err
			}
		}

		app.bot = tghbot.NewBot(options, client, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.String("gh.token")},
		), tghbot.WithLogger(app.logger))
		app.bot.SetupDispatcher(dispatcher)

		return app.bot.Run(c.Context)
	})
}

func (app *App) getEnvNames(names ...string) []string {
	return names
}

func (app *App) flags() []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config.file",
			Value:   "tghbot.yml",
			Usage:   "path to config file",
			EnvVars: app.getEnvNames("CONFIG_FILE", "CONFIG"),
		},

		// bot
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:    "bot.poll_timeout",
			Value:   10 * time.Second,
			Usage:   "Github Events API polling timeout",
			Aliases: []string{"poll_timeout"},
		}),
		altsrc.NewPathFlag(&cli.PathFlag{
			Name:    "bot.template_path",
			Usage:   "Messages templates path",
			Aliases: []string{"template_path"},
		}),

		// gh
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "gh.token",
			Required: true,
			Usage:    "Github API token",
			Aliases:  []string{"gh_token"},
			EnvVars:  app.getEnvNames("GITHUB_TOKEN"),
		}),

		// tg
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:     "tg.app_id",
			Required: true,
			Usage:    "Telegram app ID",
			Aliases:  []string{"app_id"},
			EnvVars:  app.getEnvNames("APP_ID"),
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "tg.app_hash",
			Required: true,
			Usage:    "Telegram app hash",
			Aliases:  []string{"app_hash"},
			EnvVars:  app.getEnvNames("APP_HASH"),
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "tg.bot_token",
			Required: true,
			Usage:    "Telegram bot token",
			Aliases:  []string{"token"},
			EnvVars:  app.getEnvNames("BOT_TOKEN"),
		}),
		altsrc.NewPathFlag(&cli.PathFlag{
			Name:    "tg.session_dir",
			Usage:   "Telegram session dir",
			Aliases: []string{"session_dir"},
			EnvVars: app.getEnvNames("SESSION_DIR"),
		}),
	}

	return flags
}

func (app *App) commands() []*cli.Command {
	commands := []*cli.Command{
		{
			Name:        "run",
			Description: "runs bot",
			Flags:       app.flags(),
			Action:      app.run,
		},
	}

	app.addFileConfig("config.file", commands[0])
	return commands
}

func (app *App) addFileConfig(flagName string, command *cli.Command) {
	prev := command.Before

	command.Before = func(context *cli.Context) error {
		if prev != nil {
			err := prev(context)
			if err != nil {
				return err
			}
		}

		path := context.String(flagName)
		fileContext, err := altsrc.NewYamlSourceFromFile(path)
		if err != nil {
			app.logger.Info("failed to load config from", zap.String("path", path))
			return nil
		}

		return altsrc.ApplyInputSourceValues(context, fileContext, command.Flags)
	}
}

func (app *App) cli() *cli.App {
	cliApp := &cli.App{
		Name:     "tghbot",
		Usage:    "tghbot consumes Github repo events and sends notifications to Telegram channel.",
		Commands: app.commands(),
	}

	return cliApp
}

func (app *App) Run(args []string) error {
	return app.cli().Run(args)
}

func main() {
	if err := NewApp().Run(os.Args); err != nil {
		_, _ = os.Stdout.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
