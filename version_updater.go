package main

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
)

// VersionUpdater 语言/构建系统的版本更新器接口。
// 每种语言一个实现，通过 Supported() 探测项目类型，Update() 执行版本更新。
//
// 调用链：
//
//	runVersionUpdate ──► 逐个探测 Supported()
//	      │
//	      ├─ CargoUpdater（Rust）   Supported: 存在 Cargo.toml
//	      │    └─ Update ──► 更新 Cargo.toml ──► cargo check 刷新 Cargo.lock
//	      │                                   └─► 返回 [Cargo.toml, Cargo.lock]
//	      │
//	      └─ MavenUpdater（Java）   Supported: 存在 pom.xml
//	           └─ Update ──► 更新 pom.xml 的 project 直属 <version>
//	                         └─► 返回 [pom.xml]
type VersionUpdater interface {
	// Supported 判断当前项目是否由该更新器管理
	Supported() bool
	// Update 将清单文件版本更新为 newVersion，返回需要加入暂存区的文件列表
	Update(newVersion string) ([]string, error)
}

// runVersionUpdate 版本更新调度入口：
// 遍历注册的更新器，命中第一个 Supported() 的实现执行 Update()，并将变更文件加入暂存区。
// 新增语言支持时，只需实现 VersionUpdater 接口并加入 updaters 列表。
func runVersionUpdate(updaters []VersionUpdater, newVersion string, worktree *git.Worktree) error {
	for _, updater := range updaters {
		if !updater.Supported() {
			continue
		}
		files, err := updater.Update(newVersion)
		if err != nil {
			return err
		}
		// 只暂存已被 git 跟踪的清单文件：
		// 未跟踪文件（如临时放置的测试样本 pom.xml）不属于本项目，仅更新其内容、
		// 不纳入 release 提交，避免污染仓库历史；真实项目的清单文件已在版本控制中，行为不变。
		st, err := worktree.Status()
		if err != nil {
			return err
		}
		for _, file := range files {
			fs := st.File(file)
			if fs == nil || fs.Worktree == git.Untracked {
				fmt.Printf("skipped staging untracked %s (not tracked by git)\n", file)
				continue
			}
			if _, err := worktree.Add(file); err != nil {
				return err
			}
			fmt.Printf("staged %s\n", file)
		}
		return nil
	}
	return nil
}
