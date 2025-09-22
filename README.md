
# AutoChangeLog

## 简介

AutoChangeLog 是一个用于自动为 git 仓库生成和维护 CHANGELOG.md，并自动打 tag 的工具。支持主版本、次版本、修订版本递增。

## 快速开始

### 初始化（如需 shell 版本）

```shell
change init
```

### Go 版本自动 changelog（推荐）

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

- 自动提取所有 tag 的 commit 信息，生成 CHANGELOG.md
- 自动递增版本号并打新 tag
- 支持主版本、次版本、修订版本递增
- 命令行参数友好，默认 patch

## 其他

如需兼容 shell 版本，可参考如下命令：

```shell
change
change tag
```
