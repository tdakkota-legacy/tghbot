package tghbot

import (
	"context"
	"fmt"
	"strings"

	"github.com/tdakkota/tghbot/tghbot/storage"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

type updateContext struct {
	context.Context
	telegram.UpdateClient
	users  map[int]*tg.User
	chats  map[int]*tg.Chat
	peer   tg.InputPeerClass
	fields []zap.Field
}

func (u *updateContext) Answer(m *tg.MessagesSendMessageRequest) (err error) {
	m.Peer = u.peer
	m.RandomID, err = u.RandInt64()
	if err != nil {
		return err
	}
	return u.UpdateClient.SendMessage(u, m)
}

func (b *Bot) handle(ctx updateContext, update tg.UpdateClass) error {
	ctx.fields = []zap.Field{
		zap.String("update_type", fmt.Sprintf("%T", update)),
	}

	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		return b.handleMessage(ctx, u)
	case *tg.UpdateBotInlineQuery:
		return b.handleInlineQuery(ctx, u)
	default:
		b.log.With(ctx.fields...).Info("Ignoring update")
	}
	return nil
}

func (b *Bot) handleInlineQuery(ctx updateContext, u *tg.UpdateBotInlineQuery) error {
	b.log.With(zap.String("inline_query", u.Query)).Info("Got inline query")
	return nil
}

func (b *Bot) handleMessage(ctx updateContext, u *tg.UpdateNewMessage) error {
	ctx.fields = append(ctx.fields, zap.String("message_type", fmt.Sprintf("%T", u.Message)))
	msg, ok := u.Message.(*tg.Message)
	if !ok {
		b.log.With(ctx.fields...).Info("Ignoring update")
		return nil
	}

	peerName := ""
	var peer storage.Peer
	switch p := msg.PeerID.(type) {
	case *tg.PeerUser:
		peer.PeerType = storage.User
		peer.ID = p.UserID
		ctx.peer = &tg.InputPeerUser{
			UserID: p.UserID,
		}

		peerName = ctx.users[p.UserID].Username
	case *tg.PeerChat:
		peer.PeerType = storage.Chat
		peer.ID = p.ChatID
		ctx.peer = &tg.InputPeerChat{
			ChatID: p.ChatID,
		}

		peerName = ctx.chats[p.ChatID].Title
	default:
		b.log.With(ctx.fields...).Info("Ignoring update")
		return nil
	}

	var username string
	if peer.PeerType == storage.Chat {
		from, ok := msg.FromID.(*tg.PeerUser)
		if !ok {
			b.log.With(ctx.fields...).Info(
				"Ignoring update",
				zap.String("from_id_type", fmt.Sprintf("%T", msg.FromID)),
			)
			return nil
		}

		username = ctx.users[from.UserID].Username
	} else {
		username = peerName
	}

	ctx.fields = []zap.Field{
		zap.String("message", msg.Message),
		zap.String("user", username),
		zap.String("peer", peerName),
	}

	var args []string
	for _, arg := range strings.Split(strings.TrimSpace(msg.Message), " ") {
		if len(arg) != 0 {
			args = append(args, arg)
		}
	}
	if len(args) < 1 {
		b.log.With(ctx.fields...).Info("Message is not command, ignore")
		return nil
	}

	return b.handleCommand(ctx, peer, args)
}

func (b *Bot) handleCommand(ctx updateContext, peer storage.Peer, args []string) error {
	l := b.log.With(ctx.fields...)
	command := strings.TrimSuffix(args[0], "@telghbot")
	args = args[1:]

	switch command {
	case "/addrepo":
		l.Info("Add repository command")
		if len(args) < 1 {
			return ctx.Answer(&tg.MessagesSendMessageRequest{
				Message: "/addrepo <url>",
			})
		}

		repo, err := storage.RepoFromURL(args[0])
		if err != nil {
			return ctx.Answer(&tg.MessagesSendMessageRequest{
				Message: "Некорректный URL.\nПример: https://github.com/gotd/td",
			})
		}

		err = b.storage.Add(ctx, storage.Mapping{
			Repo: repo,
			Peer: peer,
		})
		if err != nil {
			return err
		}

		return ctx.Answer(&tg.MessagesSendMessageRequest{
			Message: repo.ToGithubURL() + " добавлен",
		})
	case "/rmrepo":
		l.Info("Remove repository command")
		if len(args) < 1 {
			return ctx.Answer(&tg.MessagesSendMessageRequest{
				Message: "/rmrepo <url>",
			})
		}

		repo, err := storage.RepoFromURL(args[0])
		if err != nil {
			return ctx.Answer(&tg.MessagesSendMessageRequest{
				Message: "Некорректный URL.\nПример: https://github.com/gotd/td",
			})
		}

		err = b.storage.Remove(ctx, storage.Mapping{
			Repo: repo,
			Peer: peer,
		})
		if err != nil {
			return err
		}

		return ctx.Answer(&tg.MessagesSendMessageRequest{
			Message: repo.ToGithubURL() + " удален",
		})
	case "/listrepo":
		l.Info("List repository command")

		mappings, err := b.storage.Get(ctx, peer)
		if err != nil {
			return err
		}

		var result strings.Builder
		if len(mappings) > 0 {
			result.WriteString("Подписки:")
			for _, mapping := range mappings {
				result.WriteString(mapping.Repo.ToGithubURL())
				result.WriteByte('\n')
			}
		} else {
			result.WriteString("Ни найдено ни одной подписки")
		}

		return ctx.Answer(&tg.MessagesSendMessageRequest{
			Message: result.String(),
		})
	default:
		l.Info("Message is not command, ignore")
	}
	return nil
}
