package main

import (
	"github.com/go-git/go-git/v5/plumbing/object"
	"strconv"
	"strings"
	"time"
)

type TagCommits struct {
	Tag      string
	Version  string
	Author   string
	Date     string
	Time     time.Time
	Message  string
	Previous string
	Oldest   string
	CommitId string
	Commits  []Commit
}

type Commit struct {
	Id        string
	Author    string
	Time      time.Time
	Message   string
	Signature object.Signature
}

func fixVersion(tag string) string {
	version := strings.TrimPrefix(tag, "v")
	return version
}

// parseVersion 将版本号字符串转换为int切片
func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	// Parse parts, treating non-integer or empty parts as 0
	version := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			version = append(version, 0)
			continue
		}
		num, err := strconv.Atoi(p)
		if err != nil {
			version = append(version, 0)
			continue
		}
		version = append(version, num)
	}
	return version
}

func cmpVersion(a, b string) int {
	v1 := parseVersion(a)
	v2 := parseVersion(b)
	// compare up to the longer length, treat missing parts as 0
	maxLen := len(v1)
	if len(v2) > maxLen {
		maxLen = len(v2)
	}
	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(v1) {
			n1 = v1[i]
		}
		if i < len(v2) {
			n2 = v2[i]
		}
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}
	return 0
}

// VersionKind 版本号类型
type VersionKind int

const (
	MajorVersion VersionKind = iota // 主版本
	MinorVersion                    // 次版本
	PatchVersion                    // 默认修订版本
)

// 版本号自动加1
func incrVersion(v string, kind VersionKind) string {
	vs := parseVersion(v)
	// Ensure at least 3 parts
	for len(vs) < 3 {
		vs = append(vs, 0)
	}
	pos := int(kind)
	if pos < 0 || pos >= len(vs) {
		panic("invalid version kind")
	}
	vs[pos] = vs[pos] + 1
	// zero out lower-order parts
	for i := pos + 1; i < len(vs); i++ {
		vs[i] = 0
	}
	// build version string
	parts := make([]string, len(vs))
	for i, n := range vs {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ".")
}
