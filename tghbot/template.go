package tghbot

import (
	"text/template"
)

const TmplPR = `{{define "pr" -}}
🐽🔌 Новый pull request {{ .Repo.Name }}#{{ .PullRequest.Number }} {{ .PullRequest.Title }}
от {{ .PullRequest.User.Login }}

{{ .PullRequest.Body }}
{{end}}
`

const TmplRelease = `{{define "release" -}}
🎉 Новый релиз {{ .Repo.Name }}! {{ .Release.Name }}

{{ .Release.Body }}
{{end}}
`

const TmplPush = `{{define "push" -}}
🛠 Новые коммиты в {{ .Repo.Name }}#{{ .Ref }}

{{- range $commit := .Commits }}  
— {{ $commit.Message }} (от {{ $commit.Author.Name }} )
{{- end }}
{{end}}
`

const TmplIssue = `{{define "issue" -}}
🐛 Новый issue: {{ .Repo.Name }}#{{ .Issue.Number }} {{ .Issue.Title }}
от {{ .Issue.User.Login }}

{{ .Issue.Body }}
{{end}}
`

var builtinTemplates = map[string]string{
	"pr":      TmplPR,
	"release": TmplRelease,
	"push":    TmplPush,
	"issue":   TmplIssue,
}

func (o *Options) ParseTemplates() {
	if o.Template == nil {
		o.Template = template.New("")
	}
	for name, tmpl := range builtinTemplates {
		// not defined by user -> use builtin
		if o.Template.Lookup(name) == nil {
			template.Must(o.Template.Parse(tmpl))
		}
	}
}
