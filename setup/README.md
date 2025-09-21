# Setup - 数据库初始化

本目录包含数据库初始化相关的脚本和工具。

## 文件说明

### init_database.sql
MariaDB/MySQL 数据库初始化脚本，包含：
- 创建数据库 `solana_spl_holder`
- 创建 `spl` 表（SPL Token 信息）
- 创建 `holder` 表（持有者信息）
- 插入默认的 SPL Token 数据



## 使用方法

### 使用 SQL 脚本初始化数据库
```bash
# 连接到 MariaDB/MySQL
mysql -u root -p

# 执行初始化脚本
source /path/to/setup/init_database.sql
```

或者直接执行：
```bash
# 直接执行 SQL 脚本
mysql -u root -p < setup/init_database.sql
```

## 数据库配置

默认数据库连接配置：
- 主机：localhost:3306
- 用户名：root
- 密码：123456
- 数据库：solana_spl_holder

如需修改配置，请编辑 `init_database.sql` 脚本中的相关设置。

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