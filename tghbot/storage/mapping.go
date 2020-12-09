package storage

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

type Mapping struct {
	Repo Repo
	Peer Peer
}

type Repo struct {
	Owner string
	Name  string
}

func (r Repo) ToGithubURL() string {
	return "https://github.com" + "/" + r.Owner + "/" + r.Name
}

func RepoFromURL(rawurl string) (Repo, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return Repo{}, err
	}

	if u.Host != "github.com" {
		return Repo{}, fmt.Errorf("expected host is github.com, got %s", u.Host)
	}

	owner, name := path.Split(path.Clean(u.Path))
	if owner == "" || name == "" {
		return Repo{}, fmt.Errorf("invalid path: %s", u.Path)
	}

	owner = strings.Trim(owner, `/\`)
	name = strings.Trim(name, `/\`)
	return Repo{
		Owner: owner,
		Name:  name,
	}, nil
}

type PeerType int

const (
	Chat PeerType = iota
	Channel
	User
)

type Peer struct {
	PeerType
	ID         int
	AccessHash int64
}
