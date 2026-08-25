package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

const (
	cargoTomlFilename = "Cargo.toml"
	cargoLockFilename = "Cargo.lock"
)

// 预编译正则，避免每次调用都编译
var (
	// 匹配 [package] 或 [workspace.package] 块
	sectionRe = regexp.MustCompile(`(?m)^\s*\[(workspace\.)?package\]\s*$`)
	// 匹配 version = "x.y.z"
	versionRe = regexp.MustCompile(`(?m)^(\s*version\s*=\s*)".*"\s*$`)
)

// CargoUpdater 针对 Rust/Cargo 项目的版本更新实现。
//
// 调用链：
//
//	runVersionUpdate ──► Supported()：存在 Cargo.toml？
//	      └─ Update(newVersion)
//	            ├─ updateCargoVersion(Cargo.toml)：正则替换 [package]/[workspace.package] 中的 version
//	            ├─ 存在 Cargo.lock ──► cargo check 刷新（只更新本项目版本号，不碰依赖）
//	            └─ 返回 [Cargo.toml, Cargo.lock]
type CargoUpdater struct {
	path string // 仓库根目录（执行 cargo 命令的工作目录）
}

func NewCargoUpdater(path string) *CargoUpdater {
	return &CargoUpdater{path: path}
}

// Supported 判断当前项目是否由 Cargo 管理（存在 Cargo.toml）
func (u *CargoUpdater) Supported() bool {
	_, err := os.Stat(cargoTomlFilename)
	return err == nil
}

// Update 更新 Cargo.toml 版本号，并刷新 Cargo.lock
func (u *CargoUpdater) Update(newVersion string) ([]string, error) {
	modified, err := updateCargoVersion(cargoTomlFilename, newVersion)
	if err != nil {
		return nil, err
	}
	if !modified {
		return nil, nil
	}
	files := []string{cargoTomlFilename}
	fmt.Printf("updated %s version to %s\n", cargoTomlFilename, newVersion)
	// 运行 cargo check 同步 Cargo.lock（只更新当前项目版本号，不碰依赖）
	if _, statErr := os.Stat(cargoLockFilename); statErr == nil {
		cmd := exec.Command("cargo", "check")
		cmd.Dir = u.path
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if runErr := cmd.Run(); runErr != nil {
			fmt.Fprintf(os.Stderr, "warning: cargo check failed: %v\n", runErr)
		} else {
			files = append(files, cargoLockFilename)
			fmt.Printf("synced %s\n", cargoLockFilename)
		}
	}
	return files, nil
}

func updateCargoVersion(filePath, newVersion string) (bool, error) {
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

	inTargetSection := false
	updated := false

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// 判断是否进入目标 section
		if sectionRe.Match(trimmed) {
			inTargetSection = true
			continue
		}

		// 判断是否进入其他 section (离开目标 section)
		if bytes.HasPrefix(trimmed, []byte("[")) && inTargetSection {
			inTargetSection = false
		}

		// 如果在目标 section 内，且匹配到 version，则替换
		if inTargetSection && versionRe.Match(line) {
			// 使用正则替换，保留前面的空格和等号
			newLine := versionRe.ReplaceAll(line, fmt.Appendf(nil, `${1}"%s"`, newVersion))
			lines[i] = newLine
			updated = true
		}
	}

	if !updated {
		return false, nil
	}

	// 3. 使用原始换行符重新拼接
	newContent := bytes.Join(lines, []byte(eol))
	return true, os.WriteFile(filePath, newContent, 0644)
}
