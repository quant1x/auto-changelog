AutoChangeLog
===
# 1. shell实现的changelog
```shell
change init
change
change tag
```

# 2. go实现的changelog
### 2.1 默认, 自动提取全部tag的commit信息, 修订版本号+1, 主版本号和次版本号不变
```shell
autochangelog
```
相当于
```shell
autochangelog [patch|2]
```

### 2.2 自动提取全部tag的commit信息, 次版本号+1, 主版本号不变, 修订版本号重置为0
```shell
autochangelog [minor|1]
```

### 2.3 自动提取全部tag的commit信息, 主版本号+1, 次版本号和修订版本号重置为0
```shell
autochangelog [major|0]
```