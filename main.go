package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	changeLogFilename   = "CHANGELOG.md"
	defaultFirstVersion = "0.0.0"
)

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

// currentVersion 返回当前程序版本，由 git tag 决定，代码不硬编码版本号：
//
//  1. go install module@vX.Y.Z 安装的二进制：从模块代理构建，无 VCS 设置，
//     直接用 go 工具链嵌入的模块版本 Main.Version；
//  2. 本地 git 仓库内 go build：带 VCS 设置（vcs=git），此时 Main.Version
//     可能是伪版本（如 v1.4.4-0.20260825042314-cb7499fd4eed）或 "+dirty"，
//     一律改用 git describe 取当前分支最近可达的 tag（从 HEAD 沿祖先链回溯，
//     天然只看当前分支可达 tag，不受其他分支 tag 污染）；
//  3. 两者均不可用（如无 tag 的新仓库）时视为初始版本 0.0.0。
//
// 打新 tag 即生效，无需变更代码。
func currentVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if !hasVCSSetting(bi) {
			if v := bi.Main.Version; v != "" && v != "(devel)" {
				return v
			}
		}
	}
	return gitDescribeVersion()
}

// hasVCSSetting 判断构建时是否带 VCS 信息（本地 git 仓库内构建会有 vcs=git 等设置）
func hasVCSSetting(bi *debug.BuildInfo) bool {
	for _, s := range bi.Settings {
		if s.Key == "vcs" {
			return true
		}
	}
	return false
}

// gitDescribeVersion 用 git describe 取当前分支最近可达 tag
func gitDescribeVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	out, err := cmd.Output()
	if err != nil {
		return defaultFirstVersion
	}
	return strings.TrimSpace(string(out))
}

func main() {
	var (
		majorFlag  = flag.Bool("major", false, "主版本号+1")
		minorFlag  = flag.Bool("minor", false, "次版本号+1")
		patchFlag  = flag.Bool("patch", false, "修订版本号+1 (默认)")
		versionFlag = flag.Bool("version", false, "输出当前版本并退出")
		licenseFlag = flag.Bool("license", false, "输出第三方许可证信息并退出")
	)
	exeName := os.Args[0]
	if idx := strings.LastIndex(exeName, string(os.PathSeparator)); idx >= 0 {
		exeName = exeName[idx+1:]
	}
	flag.Usage = func() {
		fmt.Printf("Usage: %s [--major] [--minor] [--patch] [--version] [--license]\n", exeName)
		fmt.Printf("  --major   主版本号+1\n")
		fmt.Printf("  --minor   次版本号+1\n")
		fmt.Printf("  --patch   修订版本号+1 (默认)\n")
		fmt.Printf("  --version 输出当前版本并退出\n")
		fmt.Printf("  --license 输出第三方许可证信息并退出\n")
	}
	flag.Parse()
	if *licenseFlag {
		fmt.Println(noticesText)
		os.Exit(0)
	}
	if *versionFlag {
		fmt.Printf("%s %s\n", exeName, currentVersion())
		os.Exit(0)
	}
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
		fatal(err)
	}
	fmt.Println(currentPath)
	r, err := git.PlainOpen(currentPath)
	if err != nil {
		fatal(err)
	}
	// 检查工作区是否干净：存在未提交改动时，生成结果可能不准确，直接提示退出。
	// 用系统 git 判断，避免 go-git 对 CRLF/autocrlf 文件（如 change 脚本）的换行误报
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = currentPath
	statusOut, statusErr := statusCmd.Output()
	if statusErr != nil {
		fatal(statusErr)
	}
	if len(strings.TrimSpace(string(statusOut))) > 0 {
		fmt.Fprintln(os.Stderr, "working tree has uncommitted change(s); commit or stash them first")
		os.Exit(1)
	}
	worktree, err := r.Worktree()
	if err != nil {
		fatal(err)
	}
	//fmt.Printf("%+v\n", r)
	remotes, err := r.Remotes()
	if err != nil {
		fatal(err)
	}
	if len(remotes) == 0 {
		fmt.Fprintln(os.Stderr, "no remotes found in repository")
		os.Exit(1)
	}
	remote := remotes[0]
	cfg := remote.Config()
	//fmt.Printf("%+v\n", cfg)
	repositoryURL := cfg.URLs[0]
	// 获取HEAD历史记录（遍历当前分支所有可达提交，包含 merge 引入的其他分支提交）
	headRef, err := r.Head()
	if err != nil {
		fatal(err)
	}
	commitIter, err := r.Log(&git.LogOptions{From: headRef.Hash()})
	if err != nil {
		fatal(err)
	}
	var allCommits []Commit
	// 打印所有提交信息
	err = commitIter.ForEach(func(commitObj *object.Commit) error {
		commitHash := commitObj.ID()
		commit := Commit{
			Id:        commitHash.String(),
			Author:    commitObj.Committer.Name,
			Time:      commitObj.Committer.When,
			Message:   strings.TrimSpace(commitObj.Message),
			Signature: commitObj.Author, // for commit message
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
	newestCommit := allCommits[len(allCommits)-1]
	newestCommitId := newestCommit.Id
	//fmt.Printf("newestCommitId: %s\n", newestCommitId)
	//os.Exit(1)
	//fmt.Printf("commits： %+v\n", allCommits)
	// 从当前分支的 commit 中提取 tag 列表：
	// 1. 遍历仓库全部 tag，建立 commit hash -> tag 的映射
	// 2. 遍历当前分支的 commit（allCommits），取出每个 commit 上的 tag
	type TagInfo struct {
		Name   string
		Time   time.Time
		Commit *object.Commit
	}
	tagsByCommit := make(map[string][]TagInfo)
	tagIter, err := r.Tags()
	if err != nil {
		fatal(err)
	}
	_ = tagIter.ForEach(func(tagRef *plumbing.Reference) error {
		tagHash := tagRef.Hash()
		// try annotated tag first
		tagObj, err := r.TagObject(tagHash)
		if err == nil {
			annotatedCommit, _ := tagObj.Commit()
			if annotatedCommit != nil {
				tagsByCommit[annotatedCommit.ID().String()] = append(tagsByCommit[annotatedCommit.ID().String()], TagInfo{
					Name: tagRef.Name().Short(), Time: tagObj.Tagger.When, Commit: annotatedCommit,
				})
			}
			return nil
		}
		// fallback to lightweight tag (points directly to a commit)
		lightweightCommit, cerr := r.CommitObject(tagHash)
		if cerr == nil {
			tagsByCommit[lightweightCommit.ID().String()] = append(tagsByCommit[lightweightCommit.ID().String()], TagInfo{
				Name: tagRef.Name().Short(), Time: lightweightCommit.Committer.When, Commit: lightweightCommit,
			})
		}
		return nil
	})
	var tags []TagInfo
	for _, commit := range allCommits {
		if tagInfos, ok := tagsByCommit[commit.Id]; ok {
			tags = append(tags, tagInfos...)
		}
	}
	// 当前最新 commit 上已有 tag，说明该提交已发布过，无需生成新版本
	if tagInfos, ok := tagsByCommit[newestCommitId]; ok {
		names := make([]string, 0, len(tagInfos))
		for _, tagInfo := range tagInfos {
			names = append(names, tagInfo.Name)
		}
		fmt.Printf("the latest commit is already tagged (%s); nothing to do\n", strings.Join(names, ", "))
		os.Exit(0)
	}
	slices.SortFunc(tags, func(a, b TagInfo) int {
		verA := fixVersion(a.Name)
		verB := fixVersion(b.Name)
		return cmpVersion(verA, verB)
	})
	var allVersions []TagCommits

	oldest := defaultFirstVersion
	//current := defaultFirstVersion
	latest := defaultFirstVersion
	previousVersion := defaultFirstVersion
	lastTagTime := time.Unix(0, 0)

	var lastSignature object.Signature
	for _, tag := range tags {
		latest = fixVersion(tag.Name)
		if oldest == defaultFirstVersion {
			oldest = latest
		}
		tagTime := tag.Time
		tagDate := tagTime.Format(time.DateOnly)
		version := TagCommits{
			Tag:      tag.Name,
			Version:  latest,
			Previous: previousVersion,
			Date:     tagDate,
			//RepositoryURL: repositoryURL,
			Oldest: oldest,
		}
		// commit object for this tag
		tagCommit := tag.Commit
		version.Time = tagTime
		if tagCommit != nil {
			version.CommitId = tagCommit.ID().String()
		}
		version.Commits = Filter(allCommits, func(commit Commit) bool {
			commitTime := commit.Time
			inRange := commitTime.After(lastTagTime) && !commitTime.After(version.Time)
			// capture last signature of commits in range
			lastSignature = commit.Signature
			return inRange
		})
		if latest != defaultFirstVersion {
			allVersions = append(allVersions, version)
		}
		lastTagTime = version.Time
		previousVersion = latest
	}
	slices.SortFunc(allVersions, func(a, b TagCommits) int {
		return -1 * cmpVersion(a.Version, b.Version)
	})
	newVersion := incrVersion(latest, verKind)
	tag := fmt.Sprintf("v%s", newVersion)
	now := time.Now()
	version := TagCommits{
		Tag:      tag,
		Version:  newVersion,
		Previous: previousVersion,
		Date:     now.Format(time.DateOnly),
		//RepositoryURL: repositoryURL,
		Oldest: oldest,
	}
	latest = newVersion
	version.Time = now
	version.Commits = Filter(allCommits, func(commit Commit) bool {
		commitTime := commit.Time
		inRange := commitTime.After(lastTagTime) && !commitTime.After(version.Time)
		//c2 := strings.TrimSpace(commit.Message) != strings.TrimSpace(commitUpdateChangeLog)
		//return inRange && c2
		return inRange
	})
	allVersions = slices.Insert(allVersions, 0, version)
	//os.Exit(0)
	// 更新ChangeLog
	tmpl, err := template.New("ChangeLog").Parse(templateChangeLog)
	if err != nil {
		fatal(err)
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
		fatal(err)
	}
	//fmt.Println(buf.String())
	filename := changeLogFilename
	err = os.WriteFile(filename, buf.Bytes(), 0644)
	if err != nil {
		fatal(err)
	}
	_, err = worktree.Add(filename)
	if err != nil {
		fatal(err)
	}
	// 同步更新项目清单文件版本（版本更新调度，调用链见 version_updater.go）
	if err := runVersionUpdate([]VersionUpdater{
		NewCargoUpdater(currentPath),
		NewMavenUpdater(currentPath),
	}, newVersion, worktree); err != nil {
		fatal(err)
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
	commitHash, err := worktree.Commit(fmt.Sprintf("release v%s", newVersion), &git.CommitOptions{
		Author:    &lastSignature,
		Committer: &lastSignature,
	})
	if err != nil {
		fatal(err)
	}
	createdCommit, err := r.CommitObject(commitHash)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%+v\n", createdCommit)
	//err = r.Push(&git.PushOptions{})
	//if err != nil {
	//	panic(err)
	//}
	head, err := r.Head()
	if err != nil {
		fmt.Printf("get HEAD error: %s", err)
		os.Exit(1)
	}
	// 新tag
	tagMessage := fmt.Sprintf("Release version %s", newVersion)
	_, err = r.CreateTag(tag, head.Hash(), &git.CreateTagOptions{
		Message: tagMessage,
	})
	if err != nil {
		fmt.Printf("%+v\n", err)
	} else {
		fmt.Printf("new tag, %s\n", tagMessage)
		fmt.Println("Auto ChangeLog, OK.")
	}
}
