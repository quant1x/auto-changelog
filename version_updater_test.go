package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"
)

// 回归：同一仓库同时存在 Cargo.toml 与 pom.xml（如 Rust 项目内放置测试用 pom）时，
// runVersionUpdate 必须逐个执行所有 Supported() 的更新器，
// 而不是命中第一个（Cargo）后提前返回导致 pom.xml 永远不被更新。
func TestRunVersionUpdateMultiManifest(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	cargoToml := "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n"
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0-SNAPSHOT</version>
</project>
`
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	// 提交使清单文件进入版本控制（真实项目的清单文件均被 git 跟踪）
	if _, err := wt.Add("Cargo.toml"); err != nil {
		t.Fatalf("add Cargo.toml: %v", err)
	}
	if _, err := wt.Add("pom.xml"); err != nil {
		t.Fatalf("add pom.xml: %v", err)
	}
	if _, err := wt.Commit("init", &git.CommitOptions{}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	t.Chdir(dir)
	if err := runVersionUpdate([]VersionUpdater{
		NewCargoUpdater(dir),
		NewMavenUpdater(dir),
	}, "1.2.3", wt); err != nil {
		t.Fatalf("runVersionUpdate: %v", err)
	}

	gotCargo, err := os.ReadFile("Cargo.toml")
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(gotCargo), `version = "1.2.3"`) {
		t.Errorf("Cargo.toml version not updated:\n%s", gotCargo)
	}

	gotPom, err := os.ReadFile("pom.xml")
	if err != nil {
		t.Fatalf("read pom.xml: %v", err)
	}
	if !strings.Contains(string(gotPom), "<version>1.2.3</version>") {
		t.Errorf("pom.xml version not updated:\n%s", gotPom)
	}
}

// 回归：未跟踪的清单文件仅更新内容、不加入暂存区（如临时放置的测试样本 pom.xml）。
func TestRunVersionUpdateUntrackedNotStaged(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	// 仅 Cargo.toml 被跟踪；pom.xml 未跟踪（测试样本）
	cargoToml := "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n"
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0-SNAPSHOT</version>
</project>
`
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}
	if _, err := wt.Add("Cargo.toml"); err != nil {
		t.Fatalf("add Cargo.toml: %v", err)
	}
	if _, err := wt.Commit("init", &git.CommitOptions{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}

	t.Chdir(dir)
	if err := runVersionUpdate([]VersionUpdater{
		NewCargoUpdater(dir),
		NewMavenUpdater(dir),
	}, "1.2.3", wt); err != nil {
		t.Fatalf("runVersionUpdate: %v", err)
	}

	// 未跟踪的 pom.xml 内容应被更新
	gotPom, err := os.ReadFile("pom.xml")
	if err != nil {
		t.Fatalf("read pom.xml: %v", err)
	}
	if !strings.Contains(string(gotPom), "<version>1.2.3</version>") {
		t.Errorf("untracked pom.xml content not updated:\n%s", gotPom)
	}

	// 但不应进入暂存区
	st, err := wt.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	fs := st.File("pom.xml")
	if fs == nil || fs.Staging == git.Untracked {
		return
	}
	t.Errorf("untracked pom.xml should not be staged, got %v", fs.Staging)
}
