# Changelog
All notable changes to this project will be documented in this file.

{{ if .Versions -}}
<a name="unreleased"></a>
## [Unreleased]

{{ if .Unreleased.CommitGroups -}}
{{ range .Unreleased.CommitGroups -}}
{{ range .Commits -}}
- {{ .Header }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}

{{ range .Versions }}
<a name="{{ replace .Tag.Name "v"  "" 1 }}"></a>
## {{ if .Tag.Previous }}[{{ replace .Tag.Name "v" "" 1 }}]{{ else }}{{ replace .Tag.Name "v" "" 1 }}{{ end }} - {{ datetime "2006-01-02" .Tag.Date }}
### Changed
{{ range .Commits -}}
- {{ .Subject }}
{{ end -}}

{{- if .RevertCommits -}}
### Reverts
{{ range .RevertCommits -}}
- {{ .Revert.Header }}
{{ end }}
{{ end -}}

{{- if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}
{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}

{{- if .Versions }}
[Unreleased]: {{ .Info.RepositoryURL }}/compare/{{ $latest := index .Versions 0 }}{{ $latest.Tag.Name }}...HEAD
{{ range .Versions -}}
{{ if .Tag.Previous -}}
[{{ replace .Tag.Name "v" "" 1 }}]: {{ $.Info.RepositoryURL }}/compare/{{ .Tag.Previous.Name }}...{{ .Tag.Name }}
{{ end -}}
{{ end -}}
{{ end -}}