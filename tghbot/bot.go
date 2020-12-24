package tghbot

import (
	"context"
	"net/http"

	"github.com/google/go-github/v33/github"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gregjones/httpcache"
	"github.com/tdakkota/tghbot/tghbot/listener"
	"github.com/tdakkota/tghbot/tghbot/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
)

type Bot struct {
	tg      *telegram.Client
	storage storage.Storage
	subs    listener.Listener

	options Options
	log     *zap.Logger
}

func WithStorage(storage storage.Storage) func(*Bot) {
	return func(bot *Bot) {
		bot.storage = storage
	}
}

func WithLogger(log *zap.Logger) func(*Bot) {
	return func(bot *Bot) {
		bot.log = log
	}
}

func createGithubClient(src oauth2.TokenSource) *github.Client {
	var transport http.RoundTripper

	// GitHub API authentication.
	transport = &oauth2.Transport{
		Source: src,
	}

	// Memory caching.
	transport = &httpcache.Transport{
		Transport:           transport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}

	return github.NewClient(&http.Client{Transport: transport})
}

func NewBot(options Options, tg *telegram.Client, src oauth2.TokenSource, opts ...func(*Bot)) *Bot {
	options.ParseTemplates()

	b := &Bot{
		tg:      tg,
		options: options,
	}

	for _, op := range opts {
		op(b)
	}
	if b.storage == nil {
		b.storage = storage.NewInMemoryStorage()
	}
	if b.log == nil {
		b.log, _ = zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel))
	}

	b.subs = listener.NewListener(
		createGithubClient(src),
		b.storage,
		b.eventHandler,
		listener.WithLogger(b.log),
	)

	return b
}

func (b *Bot) Run(ctx context.Context) error {
	me, err := b.tg.Self(ctx)
	if err != nil {
		return err
	}
	b.log.With(zap.String("username", me.Username)).Info("getMe")
	if !me.Bot {
		b.log.Warn("expected that current user is bot")
	}

	return b.subs.Run(ctx)
}

func (b *Bot) SetupDispatcher(dispatcher tg.UpdateDispatcher) {
	dispatcher.OnNewMessage(func(ctx tg.UpdateContext, update *tg.UpdateNewMessage) error {
		return b.handleMessage(b.wrapContext(ctx), update)
	})

	dispatcher.OnBotInlineQuery(func(ctx tg.UpdateContext, update *tg.UpdateBotInlineQuery) error {
		return b.handleInlineQuery(b.wrapContext(ctx), update)
	})
}
