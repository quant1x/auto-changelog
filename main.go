package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	changeLogFilename     = "CHANGELOG.md"
	commitUpdateChangeLog = "update changelog"
	defaultFirstVersion   = "0.0.0"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Printf("OPTIONS:\n")
		fmt.Printf("\t major|0 主版本号+1\n")
		fmt.Printf("\t minor|1 次版本号+1\n")
		fmt.Printf("\t 默认修订版本号+1\n")
	}
	flag.Parse()
	verKind := PatchVersion
	argc := flag.NArg()
	versionFlag := "patch"
	if argc > 0 {
		arg := strings.TrimSpace(flag.Arg(0))
		kind := strings.ToLower(arg)
		versionFlag = kind
		switch kind {
		case "major", "0":
			verKind = MajorVersion
		case "minor", "1":
			verKind = MinorVersion
		case "patch", "2":
			verKind = PatchVersion
		default:
			verKind = PatchVersion
		}
	}
	//fmt.Printf("argc: %d, %s\n", argc, versionFlag)
	//os.Exit(0)
	_ = versionFlag
	currentPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println(currentPath)
	r, err := git.PlainOpen(currentPath)
	if err != nil {
		panic(err)
	}
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
	var allCommits []Commit
	// 打印所有提交信息
	err = cIter.ForEach(func(c *object.Commit) error {
		hash := c.ID()
		commit := Commit{
			Id:        hash.String(),
			Author:    c.Committer.Name,
			Time:      c.Committer.When,
			Message:   strings.TrimSpace(c.Message),
			Signature: c.Author, // for commit message
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
	var allVersions []TagCommits

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
		version := TagCommits{
			Tag:      obj.Name,
			Version:  latest,
			Previous: lastVersion,
			Date:     tagDate,
			//RepositoryURL: repositoryURL,
			Oldest: oldest,
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
			lastSignature = commit.Signature
			return c1
		})
		//lastSignature = obj.Tagger
		if latest != defaultFirstVersion /*&& latest != oldest*/ {
			allVersions = append(allVersions, version)
		}
		lastTime = version.Time
		lastVersion = latest
	}
	slices.SortFunc(allVersions, func(a, b TagCommits) int {
		return -1 * cmpVersion(a.Version, b.Version)
	})
	if len(allVersions) > 0 {
		lastTagCommitId := allVersions[0].CommitId
		if lastTagCommitId == lastCommitId {
			fmt.Println("tag no changed")
			os.Exit(0)
		}
	}
	newVersion := incrVersion(latest, verKind)
	tag := fmt.Sprintf("v%s", newVersion)
	now := time.Now()
	version := TagCommits{
		Tag:      tag,
		Version:  newVersion,
		Previous: lastVersion,
		Date:     now.Format(time.DateOnly),
		//RepositoryURL: repositoryURL,
		Oldest: oldest,
	}
	latest = newVersion
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
	// 更新ChangeLog
	tmpl, err := template.New("ChangeLog").Parse(templateChangeLog)
	if err != nil {
		panic(err)
	}
	data := struct {
		RepositoryURL string
		Versions      []TagCommits
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
	lastSignature.When = time.Now()
	commit, err := wt.Commit(commitUpdateChangeLog, &git.CommitOptions{
		Author:    &lastSignature,
		Committer: &lastSignature,
	})
	obj, err := r.CommitObject(commit)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", obj)
	//err = r.Push(&git.PushOptions{})
	//if err != nil {
	//	panic(err)
	//}
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
		fmt.Printf("new tag, %s\n", message)
		fmt.Println("Auto ChangeLog, OK.")
	}
}
