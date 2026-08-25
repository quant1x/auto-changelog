
# AutoChangeLog

## 简介

AutoChangeLog 是一个用于自动为 git 仓库生成和维护 CHANGELOG.md，并自动打 tag 的工具。支持主版本、次版本、修订版本递增。

支持多主版本分支并行：tag 按当前分支提取，各分支的 CHANGELOG 与版本号互不污染。低版本分支进入维护期（封版）后，可与最新分支各自独立发展，互不干扰。

## 构建

需要 Go 1.27 及以上版本：

```shell
go build -o autochangelog.exe
```

## 快速开始

在 git 仓库根目录下执行（无参数时默认修订版本号 +1）：

- 默认修订版本号 +1（patch）：

  ```shell
  autochangelog
  ```

  或

  ```shell
  autochangelog --patch
  ```

- 次版本号 +1（minor），修订号重置为 0：

  ```shell
  autochangelog --minor
  ```

- 主版本号 +1（major），次版本号和修订号重置为 0：

  ```shell
  autochangelog --major
  ```

## 功能说明

- 遍历当前分支所有可达提交，按提交提取 tag，生成 CHANGELOG.md
- tag 按当前分支隔离，多主版本分支并行互不污染
- 自动递增版本号并打新 tag；若最后一次 tag 已包含最新提交则自动跳过
- 支持主版本、次版本、修订版本递增，默认 patch
- 存在 Cargo.toml 时同步更新其版本号，并刷新 Cargo.lock

## 其他

如需兼容 shell 版本，可参考如下命令：

```shell
change
change tag
```
