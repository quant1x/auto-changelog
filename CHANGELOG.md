# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]

## [1.4.19] - 2026-08-26
### Changed
- 添加 --allow-dirty 开关：允许在存在未提交改动时继续发版（默认关闭）
- release version 1.4.19

## [1.4.18] - 2026-08-26
### Changed
- 修复 Maven pom.xml 版本更新失效：跨行 project 标签无法识别

将按行正则解析改为流式跨行 XML 标签扫描器，正确处理跨行标签、
属性值引号、多行注释与 XML 声明，避免真实项目 pom 中多行
<project> 标签导致版本更新静默失效。新增对应回归测试。
- release version 1.4.18

## [1.4.17] - 2026-08-26
### Changed
- 收敛 release 版本消息模板与 tag 名拼接，消除重复代码
- release version 1.4.17

## [1.4.16] - 2026-08-26
### Changed
- 修改发布版本的消息格式
- release version 1.4.16

## [1.4.15] - 2026-08-26
### Changed
- 将 --full 参数更名为 --refresh，用于仅重写 CHANGELOG.md 预览模板修订
- release v1.4.15

## [1.4.14] - 2026-08-26
### Changed
- 恢复旧模板
- release v1.4.14

## [1.4.13] - 2026-08-26
### Changed
- 修复最新版本段缺失 release 提交记录的问题，生成前手动追加本次将创建的 release commit
- release v1.4.13

## [1.4.12] - 2026-08-26
### Changed
- 移除 CHANGELOG 模板中重复的 ### Changed 标题，消除 MD024 重复标题告警
- release v1.4.12

## [1.4.11] - 2026-08-26
### Changed
- 修复 CHANGELOG 模板排版，标题与列表间补充空白行以符合 markdownlint 规范
- release v1.4.11

## [1.4.10] - 2026-08-26
### Changed
- 添加 markdownlint 配置，排除非手写的生成文件
- 修正 markdownlint 配置为 cli2 格式，使 ignores 排除生成文件生效
- release v1.4.10

## [1.4.9] - 2026-08-26
### Changed
- 新增项目级规则 AGENTS.md（版本策略、提交规范、许可证合规约定）
- release v1.4.9

## [1.4.8] - 2026-08-26
### Changed
- Remove legacy shell change script and update README
- release v1.4.8

## [1.4.7] - 2026-08-26
### Changed
- Add vcpkg license notice generator driven by vcpkg.json manifest dependencies
- release v1.4.7

## [1.4.6] - 2026-08-25
### Changed
- 新增 --license 参数：输出第三方许可证信息，保障单一可执行文件合规

- 用 google/go-licenses 收集编译进二进制的依赖许可证，生成 third_party/NOTICE.txt
  （含各依赖版本、SPDX 许可证标识与许可证全文）
- go:embed 将 NOTICE.txt 嵌入二进制，--license 运行时输出
- 新增 license_notice.go 与 NOTICE 生成模板 third_party/notice.tmpl
- README 补充 --license 用法与重新生成/合规检查命令
- 第三方许可证声明补充 Copyright 与 NOTICE 字段，满足严格法务审计要求

- 新增 tools/noticegen：自研生成器（纯标准库），因 go-licenses report 模板
  数据源缺少 Copyright/NoticeText 字段
  - 模块清单取自 go list -deps ./...，与实际编译产物一致
  - 提取许可证全文、版权声明行（LICENSE 无版权行时从源码头部兜底提取）、
    NOTICE 文件内容（Apache-2.0 第 4(d) 条要求）
  - 许可证类型按文本关键词识别 SPDX 标识，支持混合许可（AND 连接）
- third_party/notice.tmpl 更新为含 Copyright/NoticeText/LicenseText 判空的标准模板
- 重新生成 third_party/NOTICE.txt：21 个模块均含版权归属与许可证全文
- README 更新生成与合规检查命令
- Add markdown NOTICE.md and blank line after module name in third-party notices
- release v1.4.6

## [1.4.5] - 2026-08-25
### Changed
- 更新 README：新增 --version 用法，补充工作区检查与 Maven 递归支持说明
- 修复 --version 本地构建输出 Go 推导伪版本的问题

- 通过 BuildInfo.Settings 中的 vcs 设置区分构建方式
- 本地 git 仓库内构建（带 vcs=git）：一律用 git describe 取当前分支最近可达 tag
- go install 安装（无 VCS 设置）：使用 Go 工具链嵌入的模块版本
- release v1.4.5

## [1.4.4] - 2026-08-25
### Changed
- 新增 --version 参数：输出当前版本，版本号由 git tag 决定不硬编码

- 优先读取 go 工具链嵌入二进制的模块版本（go install module@vX.Y.Z 场景）
- 本地构建/仓库内运行时用 git describe --tags --abbrev=0 取当前分支最近可达 tag
- 裁剪本地 VCS 注入的 "+dirty" 后缀，保持纯 tag 输出
- 无 tag 时输出初始版本 0.0.0
- release v1.4.4

## [1.4.3] - 2026-08-25
### Changed
- 新增 Java Maven 版本更新支持，聚合 POM 按约定路径递归级联

- 新增 version_updater_maven.go：实现 VersionUpdater 接口
  - 更新根 pom.xml 的 project 直属 <version>
  - <module> 路径按 pom 声明解析（相对各自 pom 目录），不假设固定目录
  - BFS 递归遍历嵌套聚合模块，级联同步各模块 <parent> 版本
  - <parent> GAV 匹配直接父 pom，嵌套场景版本同步正确
  - 用 XML 栈层级只识别 <project><modules> 直属模块，避免误抓 profile
- main.go 启用 MavenUpdater
- version_updater.go 更新调用链注释
- release v1.4.3

## [1.4.2] - 2026-08-25
### Changed
- 抽象版本更新流程：新增VersionUpdater接口与调度入口，Cargo实现独立为version_updater_cargo.go，预留Java Maven扩展
- release v1.4.2

## [1.4.1] - 2026-08-25
### Changed
- 调整main.go变量命名，使其更符合语义，无逻辑改动
- 新增工作区状态检查：有未提交改动时提示退出；最新commit已有tag时提示退出（用系统git判断避免go-git的CRLF误报）
- release v1.4.1

## [1.4.0] - 2026-08-25
### Changed
- 添加 .gitignore 忽略构建产物与临时文件
- 回退 go-git 至 v5.19.2，Go 版本升级至 1.27.0
- 重构 tag 收集：从当前分支 commit 中提取 tag 列表
- 调整 imports 分组
- 修复 minor 版本递增测试期望，补充 major 用例
- 更新README，同步1.3.x新逻辑（tag按分支隔离、构建要求、Cargo同步说明）
- release v1.4.0
- README增加go install安装命令说明
- release v1.4.0

## [1.3.6] - 2026-06-18
### Changed
- refactor: replace panic with fatal for better UX
- release v1.3.6

## [1.3.5] - 2026-06-18
### Changed
- refactor: use cargo check instead of cargo generate-lockfile
- release v1.3.5

## [1.3.4] - 2026-06-18
### Changed
- feat: sync Cargo.lock after updating Cargo.toml version
- release v1.3.4

## [1.3.3] - 2026-06-18
### Changed
- refactor: optimize updateCargoVersion with pre-compiled regex and eol preservation
- release v1.3.3

## [1.3.2] - 2026-06-18
### Changed
- feat: sync Cargo.toml version when updating changelog
- release v1.3.2

## [1.3.1] - 2026-06-14
### Changed
- chore: changelog commit message now includes version number
- 更新依赖库版本
- 暂时删除CHANGELOG
- release v1.3.1

## [1.3.0] - 2026-06-14
### Changed
- chore: changelog commit message now includes version number
- 更新依赖库版本
- release v1.3.0

## [1.2.5] - 2026-01-13
### Changed
- emotes/commits 空检查、支持 lightweight tag、改进错误处理和版本解析
- update changelog

## [1.2.4] - 2026-01-13
### Changed
- 修复没有tag的情况下author为空的问题
- update changelog

## [1.2.3] - 2025-09-23
### Changed
- 重新梳理文档
- update changelog

## [1.2.2] - 2025-09-23
### Changed
- 新增控制台参数，默认patch版本号+1
- update changelog

## [1.2.1] - 2025-09-13
### Changed
- 初始化的项目可以继续后面的tag流程
- update changelog

## [1.2.0] - 2025-09-03
### Changed
- 更新依赖库版本
- update changelog

## [1.1.18] - 2025-09-03
### Changed
- 更新go最低版本到1.25
- update changelog

## [1.1.17] - 2025-05-05
### Changed
- 实验push代码
- update changelog
- update changelog
- update changelog
- 屏蔽实验性质的push代码, 更新依赖库版本
- update changelog

## [1.1.16] - 2025-05-05
### Changed
- 实验push代码
- update changelog

## [1.1.15] - 2025-02-15
### Changed
- 屏蔽测试代码
- update changelog

## [1.1.14] - 2025-02-15
### Changed
- 增加现实控制参数的测试代码

## [1.1.13] - 2025-02-15
### Changed
- 调整部分代码
- update changelog

## [1.1.12] - 2025-02-08
### Changed
- 更新依赖库版本
- update changelog
- update changelog

## [1.1.11] - 2025-02-08
### Changed
- 调整update changelog的提交时间
- update changelog

## [1.1.10] - 2025-02-08
### Changed
- 调整tag提交信息结构体名
- update changelog

## [1.1.9] - 2025-02-08
### Changed
- 删除部分废弃的代码
- update changelog

## [1.1.8] - 2025-02-08
### Changed
- 更新依赖库版本
- 优化部分代码

## [1.1.7] - 2024-08-07
### Changed
- 更新依赖库版本
- update changelog
- update changelog

## [1.1.6] - 2024-06-20
### Changed
- add LICENSE.

Signed-off-by: 王布衣 <wangfengxy@sina.cn>

## [1.1.5] - 2024-06-20
### Changed
- 修复自动更新ChangeLog的commit信息时间非最新时间的bug
- update changelog
- update changelog

## [1.1.4] - 2024-06-17
### Changed
- 修订章节编号

## [1.1.3] - 2024-06-17
### Changed
- 修订shell脚本change的用法
- update changelog

## [1.1.2] - 2024-06-17
### Changed
- 修订README, 输出两种ChangeLog工具的用法
- update changelog

## [1.1.1] - 2024-06-14
### Changed
- 增加usage
- 调整usage
- update changelog

## [1.1.0] - 2024-06-14
### Changed
- 版本号类型增加注释, 测试次版本号+1
- update changelog

## [1.0.39] - 2024-06-14
### Changed
- 允许传入参数,指定主版本,次版本及修订版本号加1, 默认是修订版本号
- update changelog

## [1.0.38] - 2024-06-02
### Changed
- 修订latest最新版本跟随新tag
- update changelog

## [1.0.37] - 2024-06-02
### Changed
- 修订模板
- update changelog

## [1.0.36] - 2024-06-02
### Changed
- 修订模板
- update changelog

## [1.0.35] - 2024-06-02
### Changed
- 删除冗余的资源url变量字段
- 增加控制台输出新tag的信息
- update changelog

## [1.0.34] - 2024-06-02
### Changed
- 去掉冗余的资源url变量字段
- update changelog

## [1.0.33] - 2024-06-02
### Changed
- 拆分模板常量
- update changelog

## [1.0.32] - 2024-06-01
### Changed
- 调整控制台展示信息
- update changelog

## [1.0.31] - 2024-06-01
### Changed
- 屏蔽git push的操作
- update changelog

## [1.0.30] - 2024-06-01
### Changed
- 修订README
- update changelog

## [1.0.29] - 2024-06-01
### Changed
- 修订README
- update changelog

## [1.0.28] - 2024-06-01
### Changed
- 调整版本无变化时控制台输出的提示信息
- update changelog

## [1.0.27] - 2024-06-01
### Changed
- 修订README-5
- update changelog

## [1.0.26] - 2024-06-01
### Changed
- 修订README-4
- update changelog
- update changelog

## [1.0.25] - 2024-06-01
### Changed
- update changelog

## [1.0.24] - 2024-06-01
### Changed
- 修订README-3
- update changelog

## [1.0.23] - 2024-06-01
### Changed
- update changelog

## [1.0.22] - 2024-06-01
### Changed
- 修订README-2
- update changelog

## [1.0.21] - 2024-06-01
### Changed
- update changelog

## [1.0.20] - 2024-06-01
### Changed
- 修订README
- update changelog

## [1.0.19] - 2024-06-01
### Changed
- update changelog

## [1.0.18] - 2024-06-01
### Changed
- 修订README
- update changelog

## [1.0.17] - 2024-06-01
### Changed
- update changelog

## [1.0.16] - 2024-06-01
### Changed
- 修订README
- update changelog

## [1.0.15] - 2024-06-01
### Changed
- update changelog

## [1.0.14] - 2024-06-01
### Changed
- 优化自动新增修订版本号的changelog逻辑
- update changelog

## [1.0.13] - 2024-06-01
### Changed
- 修订README,2
- update changelog

## [1.0.12] - 2024-06-01
### Changed
- update changelog

## [1.0.11] - 2024-06-01
### Changed
- 修订README,1
- update changelog

## [1.0.10] - 2024-06-01
### Changed
- update changelog

## [1.0.9] - 2024-06-01
### Changed
- 释放出更新changelog的commit日志
- 新增打开本地仓库的错误判断
- update changelog

## [1.0.8] - 2024-06-01
### Changed
- update changelog

## [1.0.7] - 2024-06-01
### Changed
- 新增自动增加修订版本号的逻辑
- update changelog

## [1.0.6] - 2024-06-01
### Changed
- update changelog

## [1.0.5] - 2024-06-01
### Changed
- update changelog

## [1.0.4] - 2024-06-01
### Changed
- 去掉changelog的更新日志
- 新增判断是否有新提交的情况
- update changelog
- update changelog
- update changelog

## [1.0.3] - 2024-06-01
### Changed
- tag按照版本规则排序
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog

## [1.0.2] - 2024-06-01
### Changed
- 修订git-chglog配置文件
- 新增纯go语言实现的自动changelog功能
- update changelog
- update changelog
- update changelog

## [1.0.1] - 2024-05-28
### Changed
- 增加chglog模板
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog
- update changelog

## [1.0.0] - 2024-05-28
### Changed
- add main application
- add README


[Unreleased]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.19...HEAD
[1.4.19]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.18...v1.4.19
[1.4.18]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.17...v1.4.18
[1.4.17]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.16...v1.4.17
[1.4.16]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.15...v1.4.16
[1.4.15]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.14...v1.4.15
[1.4.14]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.13...v1.4.14
[1.4.13]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.12...v1.4.13
[1.4.12]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.11...v1.4.12
[1.4.11]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.10...v1.4.11
[1.4.10]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.9...v1.4.10
[1.4.9]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.8...v1.4.9
[1.4.8]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.7...v1.4.8
[1.4.7]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.6...v1.4.7
[1.4.6]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.5...v1.4.6
[1.4.5]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.4...v1.4.5
[1.4.4]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.3...v1.4.4
[1.4.3]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.2...v1.4.3
[1.4.2]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.1...v1.4.2
[1.4.1]: https://gitee.com/quant1x/autochangelog.git/compare/v1.4.0...v1.4.1
[1.4.0]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.6...v1.4.0
[1.3.6]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.5...v1.3.6
[1.3.5]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.4...v1.3.5
[1.3.4]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.3...v1.3.4
[1.3.3]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.2...v1.3.3
[1.3.2]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.1...v1.3.2
[1.3.1]: https://gitee.com/quant1x/autochangelog.git/compare/v1.3.0...v1.3.1
[1.3.0]: https://gitee.com/quant1x/autochangelog.git/compare/v1.2.5...v1.3.0
[1.2.5]: https://gitee.com/quant1x/autochangelog.git/compare/v1.2.4...v1.2.5
[1.2.4]: https://gitee.com/quant1x/autochangelog.git/compare/v1.2.3...v1.2.4
[1.2.3]: https://gitee.com/quant1x/autochangelog.git/compare/v1.2.2...v1.2.3
[1.2.2]: https://gitee.com/quant1x/autochangelog.git/compare/v1.2.1...v1.2.2
[1.2.1]: https://gitee.com/quant1x/autochangelog.git/compare/v1.2.0...v1.2.1
[1.2.0]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.18...v1.2.0
[1.1.18]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.17...v1.1.18
[1.1.17]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.16...v1.1.17
[1.1.16]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.15...v1.1.16
[1.1.15]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.14...v1.1.15
[1.1.14]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.13...v1.1.14
[1.1.13]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.12...v1.1.13
[1.1.12]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.11...v1.1.12
[1.1.11]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.10...v1.1.11
[1.1.10]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.9...v1.1.10
[1.1.9]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.8...v1.1.9
[1.1.8]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.7...v1.1.8
[1.1.7]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.6...v1.1.7
[1.1.6]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.5...v1.1.6
[1.1.5]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.4...v1.1.5
[1.1.4]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.3...v1.1.4
[1.1.3]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.2...v1.1.3
[1.1.2]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.1...v1.1.2
[1.1.1]: https://gitee.com/quant1x/autochangelog.git/compare/v1.1.0...v1.1.1
[1.1.0]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.39...v1.1.0
[1.0.39]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.38...v1.0.39
[1.0.38]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.37...v1.0.38
[1.0.37]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.36...v1.0.37
[1.0.36]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.35...v1.0.36
[1.0.35]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.34...v1.0.35
[1.0.34]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.33...v1.0.34
[1.0.33]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.32...v1.0.33
[1.0.32]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.31...v1.0.32
[1.0.31]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.30...v1.0.31
[1.0.30]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.29...v1.0.30
[1.0.29]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.28...v1.0.29
[1.0.28]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.27...v1.0.28
[1.0.27]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.26...v1.0.27
[1.0.26]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.25...v1.0.26
[1.0.25]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.24...v1.0.25
[1.0.24]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.23...v1.0.24
[1.0.23]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.22...v1.0.23
[1.0.22]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.21...v1.0.22
[1.0.21]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.20...v1.0.21
[1.0.20]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.19...v1.0.20
[1.0.19]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.18...v1.0.19
[1.0.18]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.17...v1.0.18
[1.0.17]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.16...v1.0.17
[1.0.16]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.15...v1.0.16
[1.0.15]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.14...v1.0.15
[1.0.14]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.13...v1.0.14
[1.0.13]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.12...v1.0.13
[1.0.12]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.11...v1.0.12
[1.0.11]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.10...v1.0.11
[1.0.10]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.9...v1.0.10
[1.0.9]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.8...v1.0.9
[1.0.8]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.7...v1.0.8
[1.0.7]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.6...v1.0.7
[1.0.6]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.5...v1.0.6
[1.0.5]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.4...v1.0.5
[1.0.4]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.3...v1.0.4
[1.0.3]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.2...v1.0.3
[1.0.2]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.1...v1.0.2
[1.0.1]: https://gitee.com/quant1x/autochangelog.git/compare/v1.0.0...v1.0.1

[1.0.0]: https://gitee.com/quant1x/autochangelog.git/releases/tag/v1.0.0
