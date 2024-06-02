package main

const (
	templateChangeLog = `# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

{{- range .Versions}}
## [{{.Version}}] - {{.Date}}
### Changed
{{- range .Commits}}
- {{.Message}}
{{- end}}
{{ end}}

[Unreleased]: {{$.RepositoryURL}}/compare/v{{.Latest}}...HEAD
{{- range .Versions}}
{{if ne .Version .Oldest}}[{{.Version}}]: {{$.RepositoryURL}}/compare/v{{.Previous}}...v{{.Version}}{{- end}}
{{- end}}
[{{.Oldest}}]: {{$.RepositoryURL}}/releases/tag/v{{.Oldest}}
`
)
