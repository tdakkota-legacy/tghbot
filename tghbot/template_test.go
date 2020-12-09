package tghbot

import (
	"os"
	"testing"

	"github.com/google/go-github/v33/github"
	"github.com/stretchr/testify/require"
)

func TestTemplate(t *testing.T) {
	o := Options{}
	o.ParseTemplates()

	title := "PR title"
	body := `Body`
	username := "testuser"
	reponame := "testrepo"
	err := o.Template.ExecuteTemplate(os.Stdout, "pr", &github.PullRequestEvent{
		PullRequest: &github.PullRequest{
			Number: new(int),
			Title:  &title,
			Body:   &body,
			User: &github.User{
				Login: &username,
			},
		},
		Repo: &github.Repository{
			Name: &reponame,
		},
	})
	require.NoError(t, err)
}
