# Setup - 数据库初始化

本目录包含数据库初始化相关的脚本和工具。

## 文件说明

### init_database.sql
MariaDB/MySQL 数据库初始化脚本，包含：
- 创建数据库 `solana_spl_holder`
- 创建 `spl` 表（SPL Token 信息）
- 创建 `holder` 表（持有者信息）
- 插入默认的 SPL Token 数据

### init_spl_data.go
Go 语言编写的 SPL Token 数据初始化工具，功能：
- 连接数据库
- 批量插入/更新 SPL Token 数据
- 支持重复运行（使用 ON DUPLICATE KEY UPDATE）

## 使用方法

### 方法一：使用 SQL 脚本
```bash
# 连接到 MariaDB/MySQL
mysql -u root -p

# 执行初始化脚本
source /path/to/setup/init_database.sql
```

### 方法二：使用 Go 工具
```bash
# 进入 setup 目录
cd setup

# 运行初始化工具
go run init_spl_data.go
```

## 数据库配置

默认数据库连接配置：
- 主机：localhost:3306
- 用户名：root
- 密码：123456
- 数据库：solana_spl_holder

如需修改配置，请编辑 `init_spl_data.go` 中的 `dbConn` 变量。

## 默认 SPL Token 列表

| Symbol | Mint Address |
|--------|-------------|
| TSLAx  | XsDoVfqeBukxuZHWhdvWHBhgEHjGNst4MLodqsJHzoB |
| AAPLx  | XsbEhLAtcf6HdfpFZ5xEMdqW8nfAvcsP5bdudRLJzJp |
| NVDAx  | Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh |
| AMZNx  | Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg |
| COINx  | Xs7ZdzSHLU9ftNJsii5fCeJhoRWSC32SQGzGQtePxNu |
| HOODx  | XsvNBAYkrDRNhA7wPHQfX3ZUXZyZLdnCQDfHZ56bzpg |
| GOOGLx | XsCPL9dNWBMvFtTmwcCA5v3xWPSMEBCszbQdiLLq6aN |