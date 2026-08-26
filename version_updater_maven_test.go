package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"
)

// 真实典型 Spring Boot 项目 pom.xml：<project> 标签跨多行属性 + 外部 parent + 直属 version
const realWorldPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
        <relativePath/>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0.0</version>
    <name>demo</name>
    <description>Demo project</description>
    <properties>
        <java.version>17</java.version>
    </properties>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
`

func TestUpdatePomProjectVersion(t *testing.T) {
	cases := map[string]struct {
		content    string
		wantUpdate bool
	}{
		"single-module": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0.0</version>
</project>
`,
			wantUpdate: true,
		},
		// 回归：<project> 标签跨多行属性（真实项目常见写法），此前 scanLineTags 按行正则无法识别
		"project-tag-multiline": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0.0</version>
</project>
`,
			wantUpdate: true,
		},
		"with-parent-direct-version": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0.0</version>
</project>
`,
			wantUpdate: true,
		},
		"inherit-parent-no-version": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>com.example</groupId>
        <artifactId>parent</artifactId>
        <version>1.0.0</version>
    </parent>
    <artifactId>demo</artifactId>
</project>
`,
			wantUpdate: false,
		},
		// 多行注释不应被解析为标签污染栈
		"multiline-comment-before-project": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<!--
    Multi line comment
    about this project
-->
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>1.0.0</version>
</project>
`,
			wantUpdate: true,
		},
		// 用户实际项目（org.quant1x/quant1x）：跨行 <project> + 直属 <version>1.0-SNAPSHOT，
		// 且 dependency/plugin 中也有同名 <version>，应只改 project 直属那个
		"org-quant1x-real-pom": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>org.quant1x</groupId>
    <artifactId>quant1x</artifactId>
    <version>1.0-SNAPSHOT</version>
    <packaging>jar</packaging>
    <dependencies>
        <dependency>
            <groupId>org.junit.jupiter</groupId>
            <artifactId>junit-jupiter</artifactId>
            <version>${junit.jupiter.version}</version>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-surefire-plugin</artifactId>
                <version>3.2.5</version>
            </plugin>
        </plugins>
    </build>
</project>
`,
			wantUpdate: true,
		},
		// 写法不敏感：<version> 值带前后空白
		"version-value-with-spaces": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version> 1.0.0 </version>
</project>
`,
			wantUpdate: true,
		},
		// 写法不敏感：<version> 值跨行
		"version-value-multiline": {
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>demo</artifactId>
    <version>
        1.0.0
    </version>
</project>
`,
			wantUpdate: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			pom := filepath.Join(dir, "pom.xml")
			if err := os.WriteFile(pom, []byte(c.content), 0644); err != nil {
				t.Fatal(err)
			}
			oldWD, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(oldWD)
			modified, err := updatePomProjectVersion(pomXmlFilename, "1.1.0")
			if err != nil {
				t.Fatal(err)
			}
			if modified != c.wantUpdate {
				t.Fatalf("modified=%v, want %v", modified, c.wantUpdate)
			}
		})
	}
}

// 语义化路径验证：只改 <project> 直属 <version>，
// <parent>/<dependency>/<plugin>/<properties> 等位置的同名 version 一律不动。
func TestUpdatePomProjectVersionOnlyProjectDirect(t *testing.T) {
	const pom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>
    <groupId>org.quant1x</groupId>
    <artifactId>quant1x</artifactId>
    <version>1.0-SNAPSHOT</version>
    <properties>
        <junit.jupiter.version>5.8.2</junit.jupiter.version>
    </properties>
    <dependencies>
        <dependency>
            <groupId>org.junit.jupiter</groupId>
            <artifactId>junit-jupiter</artifactId>
            <version>${junit.jupiter.version}</version>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <artifactId>maven-surefire-plugin</artifactId>
                <version>3.2.5</version>
            </plugin>
        </plugins>
    </build>
</project>
`
	dir := t.TempDir()
	pomPath := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(pom), 0644); err != nil {
		t.Fatal(err)
	}
	oldWD, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWD)

	modified, err := updatePomProjectVersion(pomXmlFilename, "1.4.20")
	if err != nil {
		t.Fatal(err)
	}
	if !modified {
		t.Fatal("应更新 project 直属 version")
	}
	after, err := os.ReadFile(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(after)
	if !strings.Contains(content, "<version>1.4.20</version>") {
		t.Fatalf("project 直属 version 未更新:\n%s", content)
	}
	if !strings.Contains(content, "<version>3.2.0</version>") {
		t.Fatalf("parent version 不应被改动:\n%s", content)
	}
	if !strings.Contains(content, "<version>${junit.jupiter.version}</version>") {
		t.Fatalf("dependency version 不应被改动:\n%s", content)
	}
	if !strings.Contains(content, "<junit.jupiter.version>5.8.2</junit.jupiter.version>") {
		t.Fatalf("properties 中的版本定义不应被改动:\n%s", content)
	}
	if !strings.Contains(content, "<version>3.2.5</version>") {
		t.Fatalf("plugin version 不应被改动:\n%s", content)
	}
}

// 端到端：真实典型 pom + git 仓库，从根目录跑 runVersionUpdate，
// 验证 pom.xml 被更新且暂存，外部 parent 版本不被误改。
func TestRunVersionUpdateEndToEnd(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	pomPath := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(realWorldPom), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("pom.xml"); err != nil {
		t.Fatal(err)
	}

	oldWD, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWD)

	if err := runVersionUpdate([]VersionUpdater{
		NewCargoUpdater(dir),
		NewMavenUpdater(dir),
	}, "1.1.0", wt); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(after)
	if !strings.Contains(content, "<version>1.1.0</version>") {
		t.Fatalf("pom.xml 未更新项目版本:\n%s", content)
	}
	if !strings.Contains(content, "<version>3.2.0</version>") {
		t.Fatalf("外部 parent 版本 3.2.0 不应被改动:\n%s", content)
	}
}

// 回归：未跟踪的清单文件（如临时放置的测试样本 pom.xml）不属于本项目，
// 版本内容应被更新（测试可见），但不得被暂存/进入 release 提交。
func TestRunVersionUpdateSkipUntrackedManifest(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	pomPath := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(realWorldPom), 0644); err != nil {
		t.Fatal(err)
	}
	// 注意：不执行 wt.Add("pom.xml")，保持其为未跟踪文件

	oldWD, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWD)

	if err := runVersionUpdate([]VersionUpdater{
		NewCargoUpdater(dir),
		NewMavenUpdater(dir),
	}, "1.1.0", wt); err != nil {
		t.Fatal(err)
	}

	// 1. 内容已更新（测试可观察到版本变化）
	after, err := os.ReadFile(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "<version>1.1.0</version>") {
		t.Fatalf("未跟踪 pom.xml 应更新版本内容:\n%s", string(after))
	}

	// 2. 未被暂存（Staging 状态仍为 Untracked，而非 Added）
	st, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	fs := st.File("pom.xml")
	if fs == nil {
		t.Fatal("pom.xml 应出现在工作区状态中")
	}
	if fs.Staging == git.Added {
		t.Fatalf("未跟踪 pom.xml 不应被暂存，当前 Staging=%v", fs.Staging)
	}
	t.Logf("PASS: untracked pom.xml updated in worktree but not staged (Staging=%v, Worktree=%v)", fs.Staging, fs.Worktree)
}
