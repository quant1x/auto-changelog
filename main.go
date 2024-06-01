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
{{if ne .Version .Oldest}}[{{.Version}}]: {{.RepositoryURL}}/compare/v{{.Previous}}...v{{.Version}}{{- end}}
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
	if err != nil {
		panic(err)
	}
	//fmt.Println(err)
	//fmt.Printf("%+v\n", r)
	remotes, err := r.Remotes()
	if err != nil {
		panic(err)
	}
	remote := remotes[0]
	cfg := remote.Config()
	//fmt.Printf("%+v\n", cfg)
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
			Author:  c.Committer.Name,
			Time:    c.Committer.When,
			Message: strings.TrimSpace(c.Message),
		}
		//fmt.Println(commit)
		allCommits = append(allCommits, commit)
		return nil
	})
	//slices.SortFunc(allCommits, func(a, b Commit) int {
	//	return int(a.Time.UnixMilli() - b.Time.UnixMilli())
	//})
	slices.Reverse(allCommits)
	lastCommit := allCommits[len(allCommits)-1]
	lastCommitId := lastCommit.Id
	//fmt.Printf("lastCommitId: %s\n", lastCommitId)
	//os.Exit(1)
	//fmt.Printf("commits： %+v\n", allCommits)
	iter, err := r.Tags()
	if err != nil {
		panic(err)
	}
	var tags []object.Tag
	_ = iter.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash()
		obj, err := r.TagObject(hash)
		if err == nil {
			tags = append(tags, *obj)
		}
		return err
	})
	slices.SortFunc(tags, func(a, b object.Tag) int {
		av := fixVersion(a.Name)
		bv := fixVersion(b.Name)
		return cmpVersion(av, bv)
	})
	allVersions := []Version{}

	oldest := defaultFirstVersion
	//current := defaultFirstVersion
	latest := defaultFirstVersion
	lastVersion := defaultFirstVersion
	lastTime := time.Unix(0, 0)

	var lastSignature object.Signature
	for _, obj := range tags {
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
			Oldest:        oldest,
		}
		c, _ := obj.Commit()
		//version.Time = c.Committer.When
		version.Time = tagTime
		version.CommitId = c.ID().String()
		//fmt.Println(c.Hash, c.ID(), c.ParentHashes)
		//fmt.Println(lastTime, version.Time)
		version.Commits = Filter(allCommits, func(commit Commit) bool {
			tm := commit.Time
			c1 := tm.After(lastTime) && !tm.After(version.Time)
			//c2 := strings.TrimSpace(commit.Message) != strings.TrimSpace(commitUpdateChangeLog)
			//return c1 && c2
			return c1
		})
		lastSignature = obj.Tagger
		if latest != defaultFirstVersion /*&& latest != oldest*/ {
			allVersions = append(allVersions, version)
		}
		lastTime = version.Time
		lastVersion = latest
	}
	slices.SortFunc(allVersions, func(a, b Version) int {
		return -1 * cmpVersion(a.Version, b.Version)
	})
	//fmt.Printf("%+v\n", allVersions)
	lastTagCommitId := allVersions[0].CommitId
	//fmt.Println(lastTagCommitId, lastCommitId)
	if lastTagCommitId == lastCommitId {
		fmt.Println("tag no changed")
		os.Exit(0)
	}
	newVersion := incrVersion(latest)
	tag := fmt.Sprintf("v%s", newVersion)
	now := time.Now()
	version := Version{
		Tag:           tag,
		Version:       newVersion,
		Previous:      lastVersion,
		Date:          now.Format(time.DateOnly),
		RepositoryURL: repositoryURL,
		Oldest:        oldest,
	}
	version.Time = now
	version.Commits = Filter(allCommits, func(commit Commit) bool {
		tm := commit.Time
		c1 := tm.After(lastTime) && !tm.After(version.Time)
		//c2 := strings.TrimSpace(commit.Message) != strings.TrimSpace(commitUpdateChangeLog)
		//return c1 && c2
		return c1
	})
	allVersions = slices.Insert(allVersions, 0, version)
	//os.Exit(0)
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
	//fmt.Println(buf.String())
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
	//err = r.Push(&git.PushOptions{})
	//fmt.Printf("%+v\n", err)
	h, err := r.Head()
	if err != nil {
		fmt.Printf("get HEAD error: %s", err)
		os.Exit(1)
	}
	// 新tag
	message := fmt.Sprintf("Release version %s", newVersion)
	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Message: message,
	})
	if err != nil {
		fmt.Printf("%+v\n", err)
	} else {
		fmt.Println("OK.")
	}
}
