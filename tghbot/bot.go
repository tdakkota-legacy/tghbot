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
	client := tg.NewClient(b.tg)
	r, err := client.UsersGetUsers(ctx, &tg.UsersGetUsersRequest{
		ID: []tg.InputUserClass{
			&tg.InputUserSelf{},
		},
	})
	if err != nil {
		return err
	}
	if len(r) > 0 {
		me, ok := r[0].(*tg.User)
		if ok {
			b.log.With(zap.String("username", me.Username)).Info("getMe")
		}
	}

	return b.subs.Run(ctx)
}

func (b *Bot) UpdateHandler(ctx context.Context, c telegram.UpdateClient, updates *tg.Updates) error {
	// This wll be required to send message back.
	users := map[int]*tg.User{}
	for _, u := range updates.Users {
		user, ok := u.(*tg.User)
		if !ok {
			continue
		}
		users[user.ID] = user
	}

	chats := map[int]*tg.Chat{}
	for _, u := range updates.Chats {
		chat, ok := u.(*tg.Chat)
		if !ok {
			continue
		}
		chats[chat.ID] = chat
	}

	for _, update := range updates.Updates {
		if err := b.handle(updateContext{
			Context:      ctx,
			UpdateClient: c,
			users:        users,
			chats:        chats,
		}, update); err != nil {
			return err
		}
	}
	return nil
}
