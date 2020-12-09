package listener

import (
	"context"

	"github.com/tdakkota/tghbot/tghbot/storage"
)

type Link struct {
	Name string
	URL  string
}

type Payload struct {
	Data  interface{}
	Links []Link
}

func (p *Payload) AddLink(name, url string) {
	p.Links = append(p.Links, Link{
		Name: name,
		URL:  url,
	})
}

type Event struct {
	Mapping storage.Mapping
	Type    string
	Payload Payload
}

type Handler func(ctx context.Context, e Event) error
