// Command noticegen 生成第三方依赖许可证声明文件 third_party/NOTICE.txt。
//
// go-licenses report 的模板数据源只有 Name/LicenseURL/LicenseName/LicensePath/Version，
// 缺少法务审计必需的 Copyright 与 NOTICE 字段，因此本工具自行从模块缓存目录提取：
//
//   - LicenseText：许可证全文
//   - Copyright：版权声明行（从 LICENSE 头部提取，满足 MIT/BSD 的版权保留义务）
//   - NoticeText：NOTICE 文件全文（满足 Apache-2.0 第 4(d) 条的 NOTICE 再分发要求）
//
// 模块清单取自"实际编译进二进制"的包（go list -deps ./...），与运行产物严格一致。
//
// 用法（在仓库根目录）：
//
//	go run ./tools/noticegen
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

const (
	mainModule   = "gitee.com/quant1x/autochangelog"
	templateFile = "third_party/notice.tmpl"
	outputFile   = "third_party/NOTICE.txt"
)

// entry 是模板数据源中的单条记录，字段名与 third_party/notice.tmpl 一一对应
type entry struct {
	Name        string
	Version     string
	LicenseName string
	Copyright   string
	LicenseText string
	NoticeText  string
}

// module 是 go list -m 解析出的模块信息
type module struct {
	path    string
	version string
	dir     string
}

func main() {
	// 1. 收集实际编译进二进制的模块（包 → 模块路径，去重）
	used := map[string]bool{}
	out, err := exec.Command("go", "list", "-deps", "-f", "{{if .Module}}{{.Module.Path}}{{end}}", "./...").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go list -deps: %v\n", err)
		os.Exit(1)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if p := strings.TrimSpace(line); p != "" && p != mainModule {
			used[p] = true
		}
	}

	// 2. 解析模块清单（path|version|dir）
	out, err = exec.Command("go", "list", "-m", "-f", "{{.Path}}|{{.Version}}|{{.Dir}}", "all").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go list -m: %v\n", err)
		os.Exit(1)
	}
	var mods []module
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			continue
		}
		if used[parts[0]] {
			mods = append(mods, module{path: parts[0], version: parts[1], dir: parts[2]})
		}
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].path < mods[j].path })

	// 3. 逐个模块提取许可证信息
	var entries []entry
	var noLicense, noCopyright, withNotice []string
	for _, m := range mods {
		licText, ok := findLicense(m.dir)
		if !ok {
			noLicense = append(noLicense, m.path)
			continue
		}
		cp := extractCopyright(licText)
		if cp == "" {
			// LICENSE 为纯许可证模板（如 Apache-2.0 官方文本）无版权行时，
			// 从源码文件头部提取版权声明兜底
			cp = extractCopyrightFromSources(m.dir)
		}
		notice := findNotice(m.dir)
		entries = append(entries, entry{
			Name:        m.path,
			Version:     m.version,
			LicenseName: detectLicenseType(licText),
			Copyright:   cp,
			LicenseText: licText,
			NoticeText:  notice,
		})
		if cp == "" {
			noCopyright = append(noCopyright, m.path)
		}
		if notice != "" {
			withNotice = append(withNotice, m.path)
		}
	}

	// 4. 渲染模板并写出
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse template: %v\n", err)
		os.Exit(1)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, entries); err != nil {
		fmt.Fprintf(os.Stderr, "render template: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputFile, []byte(buf.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	// 5. 摘要
	fmt.Printf("generated %s: %d modules\n", outputFile, len(entries))
	if len(noLicense) > 0 {
		fmt.Printf("WARN no license file: %s\n", strings.Join(noLicense, ", "))
	}
	if len(noCopyright) > 0 {
		fmt.Printf("WARN no copyright extracted: %s\n", strings.Join(noCopyright, ", "))
	}
	if len(withNotice) > 0 {
		fmt.Printf("with NOTICE file: %s\n", strings.Join(withNotice, ", "))
	}
}

// licenseFileRe 匹配常见的许可证/版权文件（不区分大小写）
var licenseFileRe = regexp.MustCompile(`(?i)^(license|licence|copying|copyright)(\..*)?$`)

// findLicense 在模块目录中查找许可证文件并返回其全文
func findLicense(dir string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if licenseFileRe.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", false
	}
	sort.Strings(names) // 稳定顺序：LICENSE 优先于 COPYING
	b, err := os.ReadFile(filepath.Join(dir, names[0]))
	if err != nil {
		return "", false
	}
	if len(b) > 1<<20 { // 单文件上限 1MB，防止异常大文件
		b = b[:1<<20]
	}
	return string(b), true
}

// copyrightRe 匹配行首的版权声明，要求含 (c)/©/年份，
// 避免误抓许可证正文中 "copyright license to reproduce" 之类的普通句子。
// 版权行示例："Copyright (c) 2015, Emir Pasic"、"Copyright 2013 Google Inc."
var copyrightRe = regexp.MustCompile(`(?im)^[ \t]*copyright[ \t]*(?:\(c\)|©|&copy;)?[ \t]*\d{4}.*$`)

// extractCopyright 从许可证文本中提取版权声明行（去重、取前 4 行）
func extractCopyright(text string) string {
	seen := map[string]bool{}
	var lines []string
	for _, m := range copyrightRe.FindAllString(text, -1) {
		l := strings.TrimSpace(m)
		if seen[l] {
			continue
		}
		seen[l] = true
		lines = append(lines, l)
		if len(lines) == 4 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

// srcExts 是参与源码头部版权扫描的常见源码文件扩展名
var srcExts = map[string]bool{
	".go": true, ".rs": true, ".c": true, ".h": true, ".cc": true,
	".cpp": true, ".hpp": true, ".java": true, ".js": true, ".ts": true,
	".py": true, ".rb": true, ".php": true, ".swift": true, ".m": true,
}

// extractCopyrightFromSources 扫描模块目录下源码文件头部（前 8 行）的版权声明，
// 用于 LICENSE 为纯许可证模板（无版权行）时的兜底，如 Apache-2.0 官方文本。
func extractCopyrightFromSources(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	seen := map[string]bool{}
	var lines []string
	for _, e := range entries {
		if e.IsDir() || !srcExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		scanned := 0
		for sc.Scan() {
			scanned++
			if scanned > 8 {
				break
			}
			l := strings.TrimSpace(sc.Text())
			l = strings.TrimSpace(strings.TrimPrefix(l, "//")) // 容忍注释前缀
			if m := copyrightRe.FindString(l); m != "" {
				m = strings.TrimSpace(m)
				if !seen[m] {
					seen[m] = true
					lines = append(lines, m)
					if len(lines) == 3 {
						break
					}
				}
			}
		}
		f.Close()
		if len(lines) == 3 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

// detectLicenseType 依据许可证文本关键词识别 SPDX 标识，支持混合许可（AND 连接）
func detectLicenseType(text string) string {
	var kinds []string
	switch {
	case strings.Contains(text, "Apache License") && strings.Contains(text, "Version 2.0"):
		kinds = append(kinds, "Apache-2.0")
	}
	switch {
	case strings.Contains(text, "MIT License"),
		strings.Contains(text, "Permission is hereby granted, free of charge") &&
			strings.Contains(text, `THE SOFTWARE IS PROVIDED "AS IS"`):
		kinds = append(kinds, "MIT")
	}
	if strings.Contains(text, "Redistribution and use in source and binary forms") {
		if strings.Contains(text, "Neither the name") {
			kinds = append(kinds, "BSD-3-Clause")
		} else {
			kinds = append(kinds, "BSD-2-Clause")
		}
	}
	if strings.Contains(text, "Mozilla Public License") {
		kinds = append(kinds, "MPL-2.0")
	}
	if strings.Contains(text, "ISC License") ||
		(strings.Contains(text, "Permission to use, copy, modify") && strings.Contains(text, "IN NO EVENT SHALL")) {
		kinds = append(kinds, "ISC")
	}
	if strings.Contains(text, "GNU GENERAL PUBLIC LICENSE") {
		kinds = append(kinds, "GPL")
	}
	if len(kinds) == 0 {
		return "See license file"
	}
	return strings.Join(kinds, " AND ")
}

// noticeFileRe 匹配 NOTICE 文件（不区分大小写）
var noticeFileRe = regexp.MustCompile(`(?i)^notice(\..*)?$`)

// findNotice 在模块目录中查找 NOTICE 文件并返回全文（Apache-2.0 再分发要求）
func findNotice(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !noticeFileRe.MatchString(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		return string(b)
	}
	return ""
}
