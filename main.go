package main

import (
	"bytes"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"
)

const (
	changeLogFilename     = "CHANGELOG.md"
	commitUpdateChangeLog = "update changelog"
	defaultFirstVersion   = "0.0.0"
	templateChangeLog     = `# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

{{ range .Versions}}
## [{{.Version}}] - {{.Date}}
### Changed
{{- range .Commits}}
- {{.Message}}
{{- end}}
{{ end}}

[Unreleased]: {{.RepositoryURL}}/compare/v{{.Latest}}...HEAD
{{- range .Versions}}
[{{.Version}}]: {{.RepositoryURL}}/compare/v{{.Previous}}...v{{.Version}}
{{- end}}
[{{.Oldest}}]: {{.RepositoryURL}}/releases/tag/v{{.Oldest}}
`
)

func main() {
	currentPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println(currentPath)
	r, err := git.PlainOpen(currentPath)
	fmt.Println(err)
	fmt.Printf("%+v\n", r)
	remotes, err := r.Remotes()
	if err != nil {
		panic(err)
	}
	remote := remotes[0]
	cfg := remote.Config()
	fmt.Printf("%+v\n", cfg)
	repositoryURL := cfg.URLs[0]
	// 获取HEAD历史记录
	ref, err := r.Head()
	if err != nil {
		panic(err)
	}
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		panic(err)
	}
	allCommits := []Commit{}
	// 打印所有提交信息
	err = cIter.ForEach(func(c *object.Commit) error {
		hash := c.ID()
		commit := Commit{
			Id:      hash.String(),
			Author:  c.Author.Name,
			Time:    c.Author.When,
			Message: strings.TrimSpace(c.Message),
		}
		allCommits = append(allCommits, commit)
		return nil
	})
	fmt.Printf("%+v\n", allCommits)
	iter, err := r.Tags()
	if err != nil {
		panic(err)
	}
	allVersions := []Version{}

	oldest := defaultFirstVersion
	//current := defaultFirstVersion
	latest := defaultFirstVersion
	lastVersion := defaultFirstVersion
	lastTime := time.Unix(0, 0)
	var lastSignature object.Signature
	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash()
		obj, err := r.TagObject(hash)
		switch err {
		case nil:
			// Tag object present
			fmt.Println("tag hashid:", obj.Target.String())
			latest = fixVersion(obj.Name)
			if oldest == defaultFirstVersion {
				oldest = latest
			}
			tagTime := obj.Tagger.When
			tagDate := tagTime.Format(time.DateOnly)
			version := Version{
				Tag:           obj.Name,
				Version:       latest,
				Previous:      lastVersion,
				Date:          tagDate,
				RepositoryURL: repositoryURL,
			}
			c, _ := obj.Commit()
			version.Time = c.Author.When
			version.Commits = Filter(allCommits, func(commit Commit) bool {
				tm := commit.Time
				return tm.After(lastTime) && !tm.After(version.Time)
			})
			lastSignature = obj.Tagger
			if lastVersion != defaultFirstVersion {
				allVersions = append(allVersions, version)
			}
			lastTime = version.Time
			lastVersion = latest
			return nil
		case plumbing.ErrObjectNotFound:
			// Not a tag object
			return nil
		default:
			// Some other error
			return err
		}
	}); err != nil {
		// Handle outer iterator error
		panic(err)
	}
	slices.SortFunc(allVersions, func(a, b Version) int {
		return -1 * cmpVersion(a.Version, b.Version)
	})
	fmt.Printf("%+v\n", allVersions)
	tmpl, err := template.New("ChangeLog").Parse(templateChangeLog)
	if err != nil {
		panic(err)
	}
	data := struct {
		RepositoryURL string
		Versions      []Version
		Latest        string
		Oldest        string
	}{
		RepositoryURL: repositoryURL,
		Versions:      allVersions,
		Latest:        latest,
		Oldest:        oldest,
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, data)
	if err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
	wt, err := r.Worktree()
	if err != nil {
		panic(err)
	}
	filename := changeLogFilename
	err = os.WriteFile(filename, buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
	_, err = wt.Add(filename)
	if err != nil {
		panic(err)
	}
	commit, err := wt.Commit(commitUpdateChangeLog, &git.CommitOptions{
		Author: &lastSignature,
	})
	obj, err := r.CommitObject(commit)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", obj)
	err = r.Push(&git.PushOptions{})
	fmt.Printf("%+v\n", err)
}
