package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const pomXmlFilename = "pom.xml"

// 预编译正则，避免每次调用都编译
var (
	// 匹配 XML 标签事件：<tag ...> / </tag> / <tag .../>（自动跳过 <?xml 声明、<!-- 注释、<!DOCTYPE）
	xmlTagRe = regexp.MustCompile(`<([/]?)([\w-]+)([^>]*)>`)
	// 匹配独立的 <version>...</version> 行，保留缩进与标签
	pomVersionRe = regexp.MustCompile(`^(\s*<version>)([^<]*)(</version>\s*)$`)
	// 匹配独立的 <module>xxx</module> 行（聚合 pom 的子模块声明）
	pomModuleRe = regexp.MustCompile(`^\s*<module>([^<]*)</module>\s*$`)
	// 匹配独立的 <groupId>xxx</groupId> / <artifactId>xxx</artifactId> 行
	pomGroupIdRe    = regexp.MustCompile(`^\s*<groupId>([^<]*)</groupId>\s*$`)
	pomArtifactIdRe = regexp.MustCompile(`^\s*<artifactId>([^<]*)</artifactId>\s*$`)
)

// pomTag XML 标签事件
type pomTag struct {
	name        string
	closing     bool
	selfClosing bool
}

// scanLineTags 解析一行中的全部标签事件
func scanLineTags(line string) []pomTag {
	ms := xmlTagRe.FindAllStringSubmatch(line, -1)
	tags := make([]pomTag, 0, len(ms))
	for _, m := range ms {
		tags = append(tags, pomTag{
			name:        m[2],
			closing:     m[1] == "/",
			selfClosing: strings.HasSuffix(m[3], "/"),
		})
	}
	return tags
}

// MavenUpdater 针对 Java/Maven 项目的版本更新实现。
// 支持聚合（multi-module）项目：根 pom 更新自身版本，并级联同步各子模块 <parent> 版本。
// 模块路径严格按 pom 中 <modules> 声明的相对路径解析（相对各自 pom 所在目录），
// 聚合可以嵌套（子模块自身也是聚合 pom），递归遍历，不假设代码位于 src 等固定目录。
//
// 调用链：
//
//	runVersionUpdate ──► Supported()：存在 pom.xml？
//	      └─ Update(newVersion)
//	            ├─ updatePomProjectVersion(根 pom)：更新 <project> 直属 <version>
//	            ├─ pomModules(pom)：读取 <project><modules> 子模块列表（按各自目录递归）
//	            ├─ collectModulePoms(根 pom)：BFS 递归收集全部模块 pom 及各自父 pom
//	            └─ 对每个模块 pom：
//	                 └─ updateModulePomVersion()：更新 <parent> 匹配直接父 GAV 的 <version>
//	                                              以及模块自身显式的 <version>
//	            └─ 返回 [pom.xml, module-a/pom.xml, module-a/sub/pom.xml, ...]
type MavenUpdater struct {
	path string // 仓库根目录（预留：执行 mvn 命令的工作目录）
}

// mavenModule 记录一个模块 pom 及其直接父 pom（用于匹配 <parent> GAV）
type mavenModule struct {
	pom    string // 模块 pom 路径（相对仓库根）
	parent string // 直接父 pom 路径
}

func NewMavenUpdater(path string) *MavenUpdater {
	return &MavenUpdater{path: path}
}

// Supported 判断当前项目是否由 Maven 管理（存在 pom.xml）
func (u *MavenUpdater) Supported() bool {
	_, err := os.Stat(pomXmlFilename)
	return err == nil
}

// Update 更新根 pom 及其聚合子模块的版本号（递归遍历嵌套聚合）
func (u *MavenUpdater) Update(newVersion string) ([]string, error) {
	files := make([]string, 0, 4)

	// 1. 更新根 pom 的 project 直属版本
	if modified, err := updatePomProjectVersion(pomXmlFilename, newVersion); err != nil {
		return nil, err
	} else if modified {
		files = append(files, pomXmlFilename)
		fmt.Printf("updated %s version to %s\n", pomXmlFilename, newVersion)
	}

	// 2. 递归收集全部模块 pom，级联同步各模块 <parent> 版本
	rootGAV := pomGAV(pomXmlFilename)
	for _, mod := range collectModulePoms(pomXmlFilename) {
		// <parent> 匹配用直接父 pom 的 GAV：嵌套聚合时，模块 parent 指向其直接父模块而非最外层根
		parentGAV := pomGAV(mod.parent)
		if parentGAV == "" {
			parentGAV = rootGAV
		}
		if modified, err := updateModulePomVersion(mod.pom, newVersion, parentGAV); err != nil {
			return nil, err
		} else if modified {
			files = append(files, mod.pom)
			fmt.Printf("updated %s version to %s\n", mod.pom, newVersion)
		}
	}
	return files, nil
}

// collectModulePoms 从根 pom 出发 BFS 递归收集全部模块 pom。
// 每个模块的路径按该 pom 中 <modules> 声明的相对路径解析（相对当前 pom 所在目录），
// 子模块自身也可能是聚合 pom，继续向下递归；visited 防止模块目录成环。
func collectModulePoms(rootPom string) []mavenModule {
	var modules []mavenModule
	visited := map[string]bool{rootPom: true}
	queue := []string{rootPom}
	for len(queue) > 0 {
		pom := queue[0]
		queue = queue[1:]
		baseDir := filepath.Dir(pom)
		for _, modDir := range pomModules(pom) {
			modPom := filepath.ToSlash(filepath.Join(baseDir, modDir, "pom.xml"))
			if visited[modPom] {
				continue
			}
			if _, err := os.Stat(modPom); err != nil {
				continue // 模块目录缺少 pom.xml，跳过
			}
			visited[modPom] = true
			modules = append(modules, mavenModule{pom: modPom, parent: pom})
			queue = append(queue, modPom)
		}
	}
	return modules
}

// pomModules 返回 pom 声明的子模块目录列表。
// 用栈追踪层级，只识别 <project><modules> 直属的 <module>，
// 避免误抓 <profile> 等条件区域中的 <modules>。模块路径相对当前 pom 所在目录。
func pomModules(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var mods []string
	var stack []string
	for _, line := range bytes.Split(content, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		for _, tg := range scanLineTags(trimmed) {
			if !tg.closing && !tg.selfClosing {
				if tg.name == "module" && len(stack) == 2 &&
					stack[0] == "project" && stack[1] == "modules" {
					if m := pomModuleRe.FindStringSubmatch(trimmed); m != nil {
						mods = append(mods, m[1])
					}
				}
				stack = append(stack, tg.name)
			} else if tg.closing {
				if len(stack) > 0 && stack[len(stack)-1] == tg.name {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}
	return mods
}

// pomGAV 返回 pom 的 groupId:artifactId。
// project 直属优先；未声明时继承 <parent> 的 GAV（如 spring-boot-starter-parent 场景）。
func pomGAV(filePath string) string {
	if g, a := extractGAV(filePath, false); g != "" && a != "" {
		return g + ":" + a
	}
	if g, a := extractGAV(filePath, true); g != "" && a != "" {
		return g + ":" + a
	}
	return ""
}

// extractGAV 提取 pom 中的 groupId/artifactId。
// inParent=true 时取 <project><parent> 内的值；否则取 <project> 直属值。
func extractGAV(filePath string, inParent bool) (g, a string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", ""
	}
	var stack []string
	for _, line := range bytes.Split(content, []byte("\n")) {
		trimmed := string(bytes.TrimSpace(line))
		if trimmed == "" {
			continue
		}
		for _, tg := range scanLineTags(trimmed) {
			if tg.closing || tg.selfClosing {
				if tg.closing && len(stack) > 0 && stack[len(stack)-1] == tg.name {
					stack = stack[:len(stack)-1]
				}
				continue
			}
			depth := len(stack)
			if inParent {
				if depth == 2 && stack[0] == "project" && stack[1] == "parent" {
					switch tg.name {
					case "groupId":
						if m := pomGroupIdRe.FindStringSubmatch(trimmed); m != nil {
							g = m[1]
						}
					case "artifactId":
						if m := pomArtifactIdRe.FindStringSubmatch(trimmed); m != nil {
							a = m[1]
						}
					}
				}
			} else {
				if depth == 1 && stack[0] == "project" {
					switch tg.name {
					case "groupId":
						if m := pomGroupIdRe.FindStringSubmatch(trimmed); m != nil {
							g = m[1]
						}
					case "artifactId":
						if m := pomArtifactIdRe.FindStringSubmatch(trimmed); m != nil {
							a = m[1]
						}
					}
				}
			}
			stack = append(stack, tg.name)
		}
	}
	return g, a
}

// updatePomProjectVersion 更新 pom.xml 中 project 直属的 <version>（聚合根/单模块场景）。
// 跳过 <parent>/<dependency>/<plugin> 等嵌套元素的版本号。
func updatePomProjectVersion(filePath, newVersion string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	// 1. 检测原始文件的换行符，防止跨平台换行符被篡改
	eol := "\n"
	if bytes.Contains(content, []byte("\r\n")) {
		eol = "\r\n"
	}

	// 2. 按行分割，保留空行
	lines := bytes.Split(content, []byte(eol))

	// 3. 逐行解析 XML 标签，用栈跟踪当前元素层级
	//    仅当 <version> 出现在 <project> 直属层级（栈 == [project]）时才替换
	var stack []string
	inProject := false
	updated := false

	for i, line := range lines {
		trimmed := string(bytes.TrimSpace(line))
		if trimmed == "" {
			continue
		}
		replaceVersion := false
		for _, tg := range scanLineTags(trimmed) {
			if !tg.closing && !tg.selfClosing {
				if tg.name == "project" {
					inProject = true
				}
				// project 直属的 <version>（进入该标签前的栈只有 project）
				if tg.name == "version" && inProject && len(stack) == 1 {
					replaceVersion = true
				}
				stack = append(stack, tg.name)
			} else if tg.closing {
				if len(stack) > 0 && stack[len(stack)-1] == tg.name {
					stack = stack[:len(stack)-1]
				}
				if tg.name == "project" {
					inProject = false
				}
			}
		}

		// 4. 替换 project 自身版本（保留缩进与标签）
		if replaceVersion {
			lines[i] = pomVersionRe.ReplaceAll(line, fmt.Appendf(nil, `${1}%s${3}`, newVersion))
			updated = true
		}
	}

	if !updated {
		return false, nil
	}

	// 5. 使用原始换行符重新拼接
	newContent := bytes.Join(lines, []byte(eol))
	return true, os.WriteFile(filePath, newContent, 0644)
}

// updateModulePomVersion 更新子模块 pom：
//   - <parent> 的 GAV 与 parentGAV（直接父 pom 的 GAV）匹配时，更新其 <version>
//   - 子模块自身显式声明的 project 直属 <version> 也更新为 newVersion
func updateModulePomVersion(filePath, newVersion, parentGAV string) (bool, error) {
	// 第一遍：提取子模块 <parent> 的 GAV，判断是否指向直接父 pom
	updateParent := false
	if g, a := extractGAV(filePath, true); g != "" && a != "" {
		updateParent = g+":"+a == parentGAV
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	eol := "\n"
	if bytes.Contains(content, []byte("\r\n")) {
		eol = "\r\n"
	}
	lines := bytes.Split(content, []byte(eol))

	// 第二遍：更新 version
	var stack []string
	inProject := false
	updated := false

	for i, line := range lines {
		trimmed := string(bytes.TrimSpace(line))
		if trimmed == "" {
			continue
		}
		replaceVersion := false
		for _, tg := range scanLineTags(trimmed) {
			if !tg.closing && !tg.selfClosing {
				if tg.name == "project" {
					inProject = true
				}
				// project 直属 <version>
				if tg.name == "version" && inProject && len(stack) == 1 {
					replaceVersion = true
				}
				// <parent> 内的 <version>（仅当 parent 指向直接父 pom）
				if tg.name == "version" && updateParent && len(stack) == 2 &&
					stack[0] == "project" && stack[1] == "parent" {
					replaceVersion = true
				}
				stack = append(stack, tg.name)
			} else if tg.closing {
				if len(stack) > 0 && stack[len(stack)-1] == tg.name {
					stack = stack[:len(stack)-1]
				}
				if tg.name == "project" {
					inProject = false
				}
			}
		}

		if replaceVersion {
			lines[i] = pomVersionRe.ReplaceAll(line, fmt.Appendf(nil, `${1}%s${3}`, newVersion))
			updated = true
		}
	}

	if !updated {
		return false, nil
	}

	newContent := bytes.Join(lines, []byte(eol))
	return true, os.WriteFile(filePath, newContent, 0644)
}
