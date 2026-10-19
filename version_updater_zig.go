package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
)

const zigZonFilename = "build.zig.zon"

// zigVersionRe 匹配 Zig 包清单里的 `.version = "x.y.z"` 赋值。
//
// 只认 `.version` 键：`.minimum_zig_version` 不会被误匹配（它以 `.minimum` 开头）。
// 调用方只替换第一个匹配：Zig 的包版本位于根结构最前面，因此 `.dependencies`
// 等更靠后的嵌套字段不会被改动。
var zigVersionRe = regexp.MustCompile(`(?m)^([ \t]*\.version\s*=\s*)"([^"]*)"`)

// ZigUpdater 针对 Zig 项目的版本更新实现。
//
// Zig 的版本号写在 `build.zig.zon` 顶层的 `.version = "x.y.z"`；`.fingerprint`
// 由包名派生、与版本无关，因此不需要同步。
//
// 调用链：
//
//	runVersionUpdate ──► Supported()：存在 build.zig.zon？
//	      └─ Update(newVersion)
//	            └─ updateZigVersion(build.zig.zon)：替换顶层 .version
//	                  └─ 返回 [build.zig.zon]
type ZigUpdater struct {
	path string // 仓库根目录（预留：以该目录为工作目录定位清单文件）
}

// NewZigUpdater 创建 Zig 版本更新器。
func NewZigUpdater(path string) *ZigUpdater {
	return &ZigUpdater{path: path}
}

// Supported 判断当前项目是否由 Zig 管理（存在 build.zig.zon）。
func (u *ZigUpdater) Supported() bool {
	_, err := os.Stat(zigZonFilename)
	return err == nil
}

// Update 更新 build.zig.zon 的版本号。
func (u *ZigUpdater) Update(newVersion string) ([]string, error) {
	modified, err := updateZigVersion(zigZonFilename, newVersion)
	if err != nil {
		return nil, err
	}
	if !modified {
		return nil, nil
	}
	fmt.Printf("updated %s version to %s\n", zigZonFilename, newVersion)
	return []string{zigZonFilename}, nil
}

// updateZigVersion 替换 build.zig.zon 中**第一个** `.version = "..."` 的值。
//
// 按字节切片重写，不按行重新拼接，因此 CRLF/LF、缩进、文件其余内容都原样保留。
func updateZigVersion(filePath, newVersion string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	loc := zigVersionRe.FindSubmatchIndex(content)
	if loc == nil {
		return false, nil
	}

	// group 1（loc[2]:loc[3]）是 `.version = ` 前缀，保留原缩进与空白；
	// group 2 是旧版本值，整体换掉。
	var out bytes.Buffer
	out.Grow(len(content) + len(newVersion))
	out.Write(content[:loc[0]])
	out.Write(content[loc[2]:loc[3]])
	out.WriteString(`"`)
	out.WriteString(newVersion)
	out.WriteString(`"`)
	out.Write(content[loc[1]:])

	if bytes.Equal(out.Bytes(), content) {
		return false, nil
	}
	return true, os.WriteFile(filePath, out.Bytes(), 0644)
}
