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
	version := make([]int, len(parts))
	for i, p := range parts {
		num, _ := strconv.Atoi(p) // 忽略错误，假设输入总是有效的
		version[i] = num
	}
	return version
}

func cmpVersion(a, b string) int {
	v1 := parseVersion(a)
	v2 := parseVersion(b)
	for i := 0; i < len(v1) && i < len(v2); i++ {
		if v1[i] < v2[i] {
			return -1
		}
		if v1[i] > v2[i] {
			return 1
		}
	}
	// 如果一个版本号比另一个短，并且前面的部分都相等，那么较短的版本号更小
	if len(v1) < len(v2) {
		return -1
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
	length := len(vs)
	if length != 3 {
		panic("invalid version")
	}
	pos := int(kind)
	patchVersion := vs[pos] + 1
	for i := pos + 1; i < length; i++ {
		vs[i] = 0
	}
	vs[pos] = patchVersion
	version := ""
	for _, v := range vs {
		version += "." + strconv.Itoa(v)
	}
	return version[1:]
}
