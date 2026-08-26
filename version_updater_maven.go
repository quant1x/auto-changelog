package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const pomXmlFilename = "pom.xml"

// MavenUpdater 针对 Java/Maven 项目的版本更新实现。
// 支持聚合（multi-module）项目：根 pom 更新自身版本，并级联同步各子模块 <parent> 版本。
// 模块路径严格按 pom 中 <modules> 声明的相对路径解析（相对各自 pom 所在目录），
// 聚合可以嵌套（子模块自身也是聚合 pom），递归遍历，不假设代码位于 src 等固定目录。
//
// 版本更新采用 encoding/xml 语义解析：直接取 <project><version> 路径替换值文本，
// 不依赖 pom.xml 的具体写法（标签单行/跨行、注释、缩进、属性顺序、版本值换行等），
// 且只改动 <version> 元素的值部分，文件其余字节原样保留。
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
// 语义化解析：只取 <project><modules> 直属 <module> 的文本值，
// 自动排除 <profile> 等条件区域中的 <modules>，不依赖 <module> 的单行写法。
func pomModules(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	dec := xml.NewDecoder(bytes.NewReader(content))
	var stack []string
	var mods []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 3 && stack[0] == "project" &&
				stack[1] == "modules" && stack[2] == "module" {
				if v := strings.TrimSpace(string(t)); v != "" {
					mods = append(mods, v)
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
// 语义化解析：取目标元素路径下的 <groupId>/<artifactId> 文本值，不依赖单行写法。
func extractGAV(filePath string, inParent bool) (g, a string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", ""
	}
	dec := xml.NewDecoder(bytes.NewReader(content))
	var stack []string
	cur := "" // 当前打开的叶子元素名（用于关联其 CharData）
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", ""
		}
		switch t := tok.(type) {
		case xml.StartElement:
			cur = t.Name.Local
			stack = append(stack, t.Name.Local)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			cur = ""
		case xml.CharData:
			v := strings.TrimSpace(string(t))
			if v == "" || cur == "" {
				continue
			}
			if inParent {
				if len(stack) == 3 && stack[0] == "project" &&
					stack[1] == "parent" && stack[2] == cur {
					switch cur {
					case "groupId":
						g = v
					case "artifactId":
						a = v
					}
				}
			} else {
				if len(stack) == 2 && stack[0] == "project" && stack[1] == cur {
					switch cur {
					case "groupId":
						g = v
					case "artifactId":
						a = v
					}
				}
			}
		}
	}
	return g, a
}

// xmlVersionEdit 记录一处 <version> 元素的值文本字节范围（含值内空白，整体替换）。
type xmlVersionEdit struct {
	start, end int
}

// versionEdits 语义化定位 XML 内容中所有"父路径匹配 paths 的 <version> 元素"的值文本范围。
// 用 encoding/xml 事件流 + InputOffset 精确计算字节位置：
//   - 不依赖 <version> 的单行/跨行写法，值内换行、注释、空白均正确处理
//   - 只匹配给定父路径下的 version（如 ["project"] 或 ["project","parent"]），
//     自动排除 <dependency>/<plugin>/<properties> 等嵌套位置的同名元素
func versionEdits(content []byte, paths [][]string) ([]xmlVersionEdit, error) {
	dec := xml.NewDecoder(bytes.NewReader(content))
	var stack []string
	var edits []xmlVersionEdit
	var cur *xmlVersionEdit // 当前打开的待更新 <version> 元素
	for {
		tokStart := dec.InputOffset()
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		tokEnd := dec.InputOffset()
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "version" && matchesStack(stack, paths) {
				cur = &xmlVersionEdit{start: -1, end: -1}
			}
			stack = append(stack, t.Name.Local)
		case xml.EndElement:
			if len(stack) > 0 && stack[len(stack)-1] == t.Name.Local {
				if t.Name.Local == "version" && cur != nil {
					if cur.end > cur.start {
						edits = append(edits, *cur)
					}
					cur = nil
				}
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if cur != nil {
				if cur.start < 0 {
					cur.start = int(tokStart)
				}
				cur.end = int(tokEnd)
			}
		}
	}
	return edits, nil
}

// matchesStack 判断当前元素栈是否命中任一目标父路径。
func matchesStack(stack []string, paths [][]string) bool {
	for _, p := range paths {
		if slices.Equal(stack, p) {
			return true
		}
	}
	return false
}

// applyEdits 从后往前将各版本值文本替换为新版本，保留文件其余字节原样。
func applyEdits(content []byte, edits []xmlVersionEdit, newVersion string) []byte {
	out := append([]byte(nil), content...)
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		out = append(out[:e.start], append([]byte(newVersion), out[e.end:]...)...)
	}
	return out
}

// updatePomProjectVersion 更新 pom.xml 中 <project> 直属 <version>（聚合根/单模块场景）。
// 语义化解析取 <project><version> 路径，跳过 <parent>/<dependency>/<plugin> 等嵌套元素的版本号。
func updatePomProjectVersion(filePath, newVersion string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	edits, err := versionEdits(content, [][]string{{"project"}})
	if err != nil {
		return false, err
	}
	if len(edits) == 0 {
		return false, nil
	}
	return true, os.WriteFile(filePath, applyEdits(content, edits, newVersion), 0644)
}

// updateModulePomVersion 更新子模块 pom：
//   - <parent> 的 GAV 与 parentGAV（直接父 pom 的 GAV）匹配时，更新其 <version>
//   - 子模块自身显式声明的 project 直属 <version> 也更新为 newVersion
func updateModulePomVersion(filePath, newVersion, parentGAV string) (bool, error) {
	// 提取子模块 <parent> 的 GAV，判断是否指向直接父 pom
	updateParent := false
	if g, a := extractGAV(filePath, true); g != "" && a != "" {
		updateParent = g+":"+a == parentGAV
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	paths := [][]string{{"project"}}
	if updateParent {
		paths = append(paths, []string{"project", "parent"})
	}
	edits, err := versionEdits(content, paths)
	if err != nil {
		return false, err
	}
	if len(edits) == 0 {
		return false, nil
	}
	return true, os.WriteFile(filePath, applyEdits(content, edits, newVersion), 0644)
}
