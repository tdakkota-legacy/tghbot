package tghbot

import (
	"fmt"
	"strings"

	"github.com/tdakkota/tghbot/tghbot/storage"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

type updateContext struct {
	tg.UpdateContext
	*telegram.Client
	peer   tg.InputPeerClass
	fields []zap.Field
}

func (u updateContext) Answer(m *tg.MessagesSendMessageRequest) (err error) {
	m.Peer = u.peer
	m.RandomID, err = u.RandInt64()
	if err != nil {
		return err
	}
	return u.Client.SendMessage(u, m)
}

func (b *Bot) wrapContext(uctx tg.UpdateContext) updateContext {
	ctx := updateContext{
		UpdateContext: uctx,
		Client:        b.tg,
	}
	return ctx
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

		peerName = ctx.Users[p.UserID].Username
	case *tg.PeerChat:
		peer.PeerType = storage.Chat
		peer.ID = p.ChatID
		ctx.peer = &tg.InputPeerChat{
			ChatID: p.ChatID,
		}

		peerName = ctx.Chats[p.ChatID].Title
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

		username = ctx.Users[from.UserID].Username
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
			result.WriteString("Не найдено ни одной подписки\nПример:\n /addrepo https://github.com/gotd/td")
		}

		return ctx.Answer(&tg.MessagesSendMessageRequest{
			Message: result.String(),
		})
	default:
		l.Info("Message is not command, ignore")
	}
	return nil
}
