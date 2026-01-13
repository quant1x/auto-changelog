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
	var (
		majorFlag = flag.Bool("major", false, "主版本号+1")
		minorFlag = flag.Bool("minor", false, "次版本号+1")
		patchFlag = flag.Bool("patch", false, "修订版本号+1 (默认)")
	)
	flag.Usage = func() {
		exeName := os.Args[0]
		if idx := strings.LastIndex(exeName, string(os.PathSeparator)); idx >= 0 {
			exeName = exeName[idx+1:]
		}
		fmt.Printf("Usage: %s [--major] [--minor] [--patch]\n", exeName)
		fmt.Printf("  --major   主版本号+1\n")
		fmt.Printf("  --minor   次版本号+1\n")
		fmt.Printf("  --patch   修订版本号+1 (默认)\n")
	}
	flag.Parse()
	verKind := PatchVersion
	if *majorFlag {
		verKind = MajorVersion
	} else if *minorFlag {
		verKind = MinorVersion
	} else if *patchFlag {
		verKind = PatchVersion
	}
	// 如果没有任何参数，默认 patch
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
	if len(remotes) == 0 {
		fmt.Fprintln(os.Stderr, "no remotes found in repository")
		os.Exit(1)
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
	if len(allCommits) == 0 {
		fmt.Fprintln(os.Stderr, "no commits found in repository; cannot create changelog")
		os.Exit(1)
	}
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
	type TagInfo struct {
		Name   string
		Time   time.Time
		Commit *object.Commit
	}
	var tags []TagInfo
	_ = iter.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash()
		// try annotated tag first
		obj, err := r.TagObject(hash)
		if err == nil {
			c, _ := obj.Commit()
			tags = append(tags, TagInfo{Name: ref.Name().Short(), Time: obj.Tagger.When, Commit: c})
			return nil
		}
		// fallback to lightweight tag (points directly to a commit)
		c, cerr := r.CommitObject(hash)
		if cerr == nil {
			tags = append(tags, TagInfo{Name: ref.Name().Short(), Time: c.Committer.When, Commit: c})
			return nil
		}
		return nil
	})
	slices.SortFunc(tags, func(a, b TagInfo) int {
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
		tagTime := obj.Time
		tagDate := tagTime.Format(time.DateOnly)
		version := TagCommits{
			Tag:      obj.Name,
			Version:  latest,
			Previous: lastVersion,
			Date:     tagDate,
			//RepositoryURL: repositoryURL,
			Oldest: oldest,
		}
		// commit object for this tag
		c := obj.Commit
		version.Time = tagTime
		if c != nil {
			version.CommitId = c.ID().String()
		}
		version.Commits = Filter(allCommits, func(commit Commit) bool {
			tm := commit.Time
			c1 := tm.After(lastTime) && !tm.After(version.Time)
			// capture last signature of commits in range
			lastSignature = commit.Signature
			return c1
		})
		if latest != defaultFirstVersion {
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
	// Ensure we have a valid signature (fallback to last commit's author when no annotated tags exist)
	if lastSignature.Name == "" && lastSignature.Email == "" {
		if len(allCommits) > 0 {
			// use the latest commit's signature as the author/committer
			lastSignature = allCommits[len(allCommits)-1].Signature
		} else {
			// No commits in repository — this tool requires at least one commit
			fmt.Fprintln(os.Stderr, "no commits found in repository; cannot create changelog")
			os.Exit(1)
		}
	}
	lastSignature.When = time.Now()
	commit, err := wt.Commit(commitUpdateChangeLog, &git.CommitOptions{
		Author:    &lastSignature,
		Committer: &lastSignature,
	})
	if err != nil {
		panic(err)
	}
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
