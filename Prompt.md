# Solana SPL Token Holder Tracker - 项目需求文档

## 项目概述

使用 Golang 开发一个高性能的 Solana SPL Token 持有者追踪工具，名称为 `solana-spl-holder`。该工具能够定期从 Solana 区块链获取 SPL Token 持有者信息，存储到 MariaDB 数据库中，并提供完整的 RESTful API 服务供查询和管理。

## 核心功能需求

### 1. 数据采集模块

#### RPC 接口调用
使用 Solana RPC 的 `getProgramAccounts` 方法获取 SPL Token 账户信息：

```bash
curl "https://api.devnet.solana.com" \
-X POST \
-H "Content-Type: application/json" \
-d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "getProgramAccounts",
    "params": [
        "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb",
        {
            "encoding": "jsonParsed",
            "filters": [
                {
                    "memcmp": {
                        "offset": 0,
                        "bytes": "指定的mint地址"
                    }
                }
            ]
        }
    ]
}'
```

#### 数据处理要求
- 解析 RPC 响应中的所有账户信息
- 提取关键字段：pubkey、owner、mint、amount、decimals、uiAmount 等
- 实现数据去重和增量更新机制
- 支持批量数据处理以提高性能

#### 定时任务
- 可配置的采集间隔（默认 300 秒）
- 支持多个 SPL Token 的并发采集
- 实现优雅的错误处理和重试机制
- 支持上下文取消和优雅关闭

### 2. 数据存储模块

#### 数据库设计
使用 MariaDB 作为主要存储引擎，包含以下表结构：

**SPL Token 配置表 (spl)**
- id: 主键
- symbol: Token 符号
- mint_address: Token 合约地址
- created_at/updated_at: 时间戳

**持有者信息表 (holder)**
- id: 主键
- mint_address: Token 合约地址
- pubkey: 持有者地址
- lamports: 账户余额（lamports）
- owner: 账户所有者
- state: 账户状态
- amount: Token 数量（原始值）
- ui_amount: Token 数量（UI 显示值）
- decimals: 小数位数
- created_at/updated_at: 时间戳

#### 数据操作
- 实现 UPSERT 操作（INSERT ... ON DUPLICATE KEY UPDATE）
- 支持事务处理确保数据一致性
- 实现连接池管理和自动重连
- 提供数据库健康检查机制

### 3. API 服务模块

#### RESTful API 端点

**核心查询接口**
- `GET /holders` - 获取持有者列表
  - 支持分页：`?page=1&limit=20`
  - 支持过滤：`?mint_address=xxx`
  - 支持排序：`?sort=ui_amount` 或 `?sort=-ui_amount`

**SPL Token 管理接口**
- `POST /api/spl` - 创建 SPL Token 配置
- `GET /api/spl` - 获取 SPL Token 列表
- `GET /api/spl/{id}` - 获取单个 SPL Token
- `PUT /api/spl/{id}` - 更新 SPL Token
- `DELETE /api/spl/{id}` - 删除 SPL Token

**系统接口**
- `GET /health` - 健康检查
- `GET /` - API 文档页面

#### 响应格式
统一的 JSON 响应格式：
```json
{
  "success": true,
  "data": [...],
  "total": 100,
  "page": 1,
  "limit": 20,
  "error": null
}
```

### 4. 命令行接口

使用 Cobra 库实现命令行参数处理：

```bash
./solana-spl-holder [flags]

Flags:
  --rpc_url string      Solana RPC 节点地址 (default "https://api.devnet.solana.com")
  --db_conn string      MariaDB 连接字符串 (default "root:123456@tcp(localhost:3306)/solana_spl_holder?charset=utf8mb4&parseTime=True&loc=Local")
  --interval_time int   数据采集间隔时间(秒) (default 300)
  --listen_port int     HTTP 服务监听端口 (default 8090)
  -h, --help           显示帮助信息
```

## 技术要求

### 1. 代码质量
- 所有代码合并到单个文件 `server/main.go` 中
- 严格检查并移除未使用的导入包
- 实现完整的错误处理和日志记录
- 使用结构化日志，包含不同级别（INFO、ERROR、DEBUG）

### 2. 性能优化
- HTTP 请求超时设置为 30 秒
- 实现连接池和资源复用
- 支持并发数据采集和处理
- 实现合理的内存管理

### 3. 可靠性保障
- 实现优雅关闭机制（捕获 SIGINT、SIGTERM 信号）
- 网络异常和数据库异常的自动重试
- 完整的错误日志记录
- 数据库连接健康检查

### 4. 开发和部署
- 提供完整的 Makefile 支持构建、测试、部署
- 支持多环境配置（devnet、localnet、mainnet）
- 包含数据库初始化脚本
- 提供完整的测试套件

## 项目结构

```
solana-spl-holder/
├── server/
│   └── main.go              # 主程序文件（所有代码）
├── setup/
│   ├── init_database.sql    # 数据库初始化脚本
│   ├── init_spl_data.go     # SPL 数据初始化工具
│   └── README.md            # 设置说明
├── test/
│   ├── api_test.go          # API 测试套件
│   └── README.md            # 测试说明
├── Makefile                 # 构建工具
├── go.mod                   # Go 模块依赖
├── devnet.sh               # 开发网络启动脚本
├── localnet.sh             # 本地网络启动脚本
├── mainnet.sh              # 主网启动脚本
└── README.md               # 项目文档
```

## 依赖库要求

- `github.com/spf13/cobra` - 命令行参数处理
- `github.com/go-sql-driver/mysql` - MariaDB 驱动
- 标准库：`net/http`、`database/sql`、`encoding/json`、`context` 等

## 质量保证

### 1. 测试覆盖
- 单元测试覆盖所有核心功能
- API 端点集成测试
- 并发安全性测试
- 性能基准测试

### 2. 错误处理
- 网络请求失败重试机制
- 数据库连接异常处理
- 数据解析错误处理
- 优雅的服务降级

### 3. 监控和日志
- 结构化日志输出
- 关键操作的性能指标
- 错误统计和告警
- 健康检查端点

## 部署和运维

### 1. 环境支持
- 支持 Docker 容器化部署
- 支持多环境配置管理
- 提供启动脚本和服务管理

### 2. 配置管理
- 支持环境变量配置
- 支持配置文件热重载
- 敏感信息安全管理

### 3. 扩展性
- 支持水平扩展
- 支持多实例负载均衡
- 支持数据库读写分离

## 安全要求

- 输入参数验证和清理
- SQL 注入防护
- 访问频率限制
- 敏感信息脱敏处理

---

**注意事项：**
1. 确保所有网络请求都有适当的超时设置
2. 避免在日志中输出敏感信息
3. 实现合理的资源限制和清理机制
4. 保持代码的可读性和可维护性
5. 遵循 Go 语言最佳实践和编码规范