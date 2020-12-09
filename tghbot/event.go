package tghbot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"
	"github.com/tdakkota/tghbot/tghbot/listener"
	"github.com/tdakkota/tghbot/tghbot/storage"
)

func (b *Bot) eventHandler(ctx context.Context, e listener.Event) error {
	return b.sendTemplate(ctx, e.Mapping.Peer, e.Type, e.Payload)
}

var errInvalidPeerType = errors.New("invalid peer type")

func (b *Bot) sendTemplate(ctx context.Context, peer storage.Peer, tmplName string, payload listener.Payload) error {
	data := payload.Data

	var s strings.Builder
	err := b.options.Template.ExecuteTemplate(&s, tmplName, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	var inputPeer tg.InputPeerClass
	switch peer.PeerType {
	case storage.Channel:
		inputPeer = &tg.InputPeerChannel{
			ChannelID:  peer.ID,
			AccessHash: peer.AccessHash,
		}
	case storage.User:
		inputPeer = &tg.InputPeerUser{
			UserID:     peer.ID,
			AccessHash: peer.AccessHash,
		}
	case storage.Chat:
		inputPeer = &tg.InputPeerChat{
			ChatID: peer.ID,
		}
	default:
		return errInvalidPeerType
	}

	var rply tg.ReplyMarkupClass
	if len(payload.Links) > 0 {
		rply := &tg.ReplyKeyboardMarkup{
			Resize: true,
		}
		for _, link := range payload.Links {
			rply.Rows = append(rply.Rows, tg.KeyboardButtonRow{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonUrl{
						Text: link.Name,
						URL:  link.URL,
					},
				},
			})
		}
	}

	randomID, err := b.tg.RandInt64()
	if err != nil {
		return err
	}
	err = b.tg.SendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:        inputPeer,
		RandomID:    randomID,
		Message:     s.String(),
		ReplyMarkup: rply,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
