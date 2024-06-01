package main

import (
	"strconv"
	"strings"
	"time"
)

type Version struct {
	Tag           string
	Version       string
	Author        string
	Date          string
	Time          time.Time
	Message       string
	Previous      string
	RepositoryURL string
	Commits       []Commit
}

type Commit struct {
	Id      string
	Author  string
	Time    time.Time
	Message string
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
