package tghbot

import (
	"text/template"
	"time"
)

type Options struct {
	PollTimeout time.Duration
	Template    *template.Template
}
