package tgutil

import "github.com/gotd/td/tg"

func ConvertPeerToInputPeer(peer tg.PeerClass) tg.InputPeerClass {
	switch v := peer.(type) {
	case *tg.PeerUser: // peerUser#9db1bc6d
		return &tg.InputPeerUser{
			UserID:     v.UserID,
			AccessHash: 0,
		}
	case *tg.PeerChat: // peerChat#bad0e5bb
		return &tg.InputPeerChat{
			ChatID: v.ChatID,
		}
	case *tg.PeerChannel: // peerChannel#bddde532
		return &tg.InputPeerChannel{
			ChannelID: v.ChannelID,
		}
	default:
		panic(v)
	}
}
