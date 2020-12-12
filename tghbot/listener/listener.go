package listener

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/google/go-github/v33/github"
	"github.com/tdakkota/tghbot/tghbot/storage"
)

type Listener struct {
	gh      *github.Client
	storage storage.Storage

	handler     Handler
	pollTimeout time.Duration
	lastUpdate  time.Time
	log         *zap.Logger
}

func WithPollTimeout(pollTimeout time.Duration) func(*Listener) {
	return func(listener *Listener) {
		listener.pollTimeout = pollTimeout
	}
}

func WithLogger(logger *zap.Logger) func(*Listener) {
	return func(listener *Listener) {
		listener.log = logger
	}
}

func NewListener(gh *github.Client, storage storage.Storage, handler Handler, opts ...func(*Listener)) Listener {
	s := Listener{
		gh:          gh,
		storage:     storage,
		handler:     handler,
		pollTimeout: 10 * time.Second,
		lastUpdate:  time.Now(),
	}

	for _, op := range opts {
		op(&s)
	}

	if s.log == nil {
		s.log, _ = zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel))
	}

	return s
}

func (s Listener) Run(ctx context.Context) error {
	timer := time.NewTimer(s.pollTimeout)

	s.log.Info("running Github API event listener")
	defer func() {
		s.log.Info("stopping Github API event listener")
	}()

	for {
		select {
		case <-timer.C:
			timer.Reset(s.pollTimeout)

			mappings, err := s.storage.List(ctx)
			if err != nil {
				return err
			}

			for _, m := range mappings {
				repo := m.Repo
				events, resp, err := s.gh.Activity.ListRepositoryEvents(ctx, repo.Owner, repo.Name, nil)
				if err != nil {
					return err
				}

				if resp.Header.Get("X-From-Cache") == "1" {
					s.log.Info("skipping due to events got from cache")
					continue
				}

				err = s.handleEvents(ctx, m, events)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Listener) handleEvents(ctx context.Context, m storage.Mapping, events []*github.Event) error {
	c := 0

	for _, event := range events {
		l := s.log.With(
			zap.String("repo", m.Repo.ToGithubURL()),
			zap.String("event_type", event.GetType()),
		)
		if event.GetCreatedAt().Before(s.lastUpdate) {
			continue
		}
		s.lastUpdate = time.Now().Add(s.pollTimeout * -1).Add(time.Second)
		c++

		l.Info("handling event")
		p, err := event.ParsePayload()
		if err != nil {
			return err
		}

		e := Event{
			Mapping: m,
			Payload: Payload{
				Data: p,
			},
		}
		repoName := m.Repo.Name
		switch payload := p.(type) {
		case *github.PullRequestEvent:
			payload.Repo = &github.Repository{
				Name: &repoName,
			}

			if payload.GetAction() == "opened" && payload.PullRequest != nil {
				e.Type = "pr"
				e.Payload.AddLink("diff", payload.PullRequest.GetDiffURL())
				return s.handler(ctx, e)
			}
		case *github.ReleaseEvent:
			payload.Repo = &github.Repository{
				Name: &repoName,
			}

			if payload.GetAction() == "published" && payload.Release != nil {
				e.Type = "release"
				e.Payload.AddLink("Релиз", payload.Release.GetURL())
				return s.handler(ctx, e)
			}
		case *github.PushEvent:
			payload.Repo = &github.PushEventRepository{
				Name: &repoName,
			}

			e.Type = "push"
			return s.handler(ctx, e)
		case *github.IssuesEvent:
			payload.Repo = &github.Repository{
				Name: &repoName,
			}

			if payload.GetAction() == "opened" && payload.Issue != nil {
				e.Type = "issue"
				e.Payload.AddLink("Issue", payload.Issue.GetURL())
				return s.handler(ctx, e)
			}
		}
	}

	if c == 0 {
		s.log.Info("No new events")
	}

	return nil
}
