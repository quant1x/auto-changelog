# AGENTS.md — autochangelog 项目规则

本文件是 AI 编码助手的项目级规则，进入本仓库工作时必须遵守。

## 项目概述

Go 编写的 changelog 自动生成工具，从 git 历史与 tag 生成 `CHANGELOG.md`。单二进制分发，无运行时外部依赖。

## 版本与分支策略（重要）

- 仓库采用**多主版本线并行**模式（类 Python/go 大版本产品线）：`1.0.x`、`1.1.x`、`1.2.x`、`1.3.x`、`1.4.x` 各自独立发展，互不同步新功能。
- **低版本线封版后不再改动代码与逻辑**，除非是 CVE/安全修复。当前主线为 `1.4.x`。
- tag 严格隔离：**低版本线的 tag 不能被高版本线污染，反之亦然**。发版前必须核对当前分支，确认 tag 归属正确。
- 新功能、重构只进当前主线（`1.4.x`），不得回写已封版分支。

## 生成 changelog 的强制要求

- 必须使用**当前目录下编译的新版** `.\autochangelog.exe`（`go build -o autochangelog.exe .` 后运行）。
- **禁止调用 PATH 中的旧版工具**（`autochangelog` / `change`），否则会生成错误内容。
- 修改版本号后必须重新编译再生成 changelog。

## 提交规范

- **commit 信息必须使用中文描述**，不得用英文或其他语言（用户明确要求）。
- PowerShell 下执行中文 `git commit -m "..."` 若遇编码问题，先执行 `chcp 65001`，或改用 `git commit -F <UTF-8 临时文件>`。
- 未明确要求时不主动提交；用户说"提交"才执行 `git add` + `git commit`。

## 第三方许可证合规

### Go 依赖（tools/noticegen）

- `go run ./tools/noticegen` 会同时生成两份：
  - `third_party/NOTICE.txt`（纯文本，`third_party/notice.tmpl`）
  - `third_party/NOTICE.md`（markdown，`third_party/notice.md.tmpl`）
- 模板格式约定：**模块名之后必须保留空白行**，再输出 `Copyright:` / `License:` 行；许可证全文放入代码块。
- 许可证识别按全文关键词（MIT/BSD/Apache-2.0/MPL/ISC/Zlib 等），支持混合许可 `AND` 连接；`License:` 行匹配必须"行首 + 带冒号"，禁止跨行误判。
- `entry` 结构体字段与模板一一对应（含 `LicenseURL`，由模块路径推导、去掉 `/vN` 主版本后缀）。

### C++ 依赖（tools/vcpkg_notices.py）

- 以项目根目录 `vcpkg.json` 的 `dependencies` 为**唯一依赖清单**，只检索声明的 port 的 `copyright`（license）与 `NOTICE` 文件，不扫描整个 installed 目录。
- 检索源优先级：`--installed-dir`（`share/<port>/copyright`）→ `--ports-dir`（`C:/vcpkg/ports/<port>/copyright`）。
- 版本号来源优先级：`vcpkg.json` → `vcpkg.status` → copyright 首行启发式。
- 输出格式与 notice.tmpl 对齐：名称后空白行、`Copyright:` / `License:`、许可证全文、NOTICE 段。

## 工具命令速查

```powershell
# 生成 changelog（必须用当前目录编译的新版）
go build -o autochangelog.exe .
.\autochangelog.exe          # 增量
.\autochangelog.exe -full    # 全量重建

# 重新生成第三方许可证声明
go run ./tools/noticegen
python tools/vcpkg_notices.py --installed-dir vcpkg_installed/x64-windows --product-name "quant1x"
```
