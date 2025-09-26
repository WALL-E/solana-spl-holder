# Test - 测试目录

本目录包含项目的测试文件和测试工具。

## 文件说明

### api_test.go
API 端点测试文件，包含：
- 健康检查端点测试
- 持有者查询端点测试（基础查询、分页、过滤、排序）
- API 文档端点测试
- 并发测试和性能测试

## 测试结构

```
test/
├── README.md          # 测试说明文档
├── api_test.go        # API 端点测试
├── unit_test.go       # 单元测试（待创建）
└── integration_test.go # 集成测试（待创建）
```

## 运行测试

### 运行所有测试
```bash
# 在项目根目录运行
go test ./test/...

# 或者进入 test 目录运行
cd test
go test .
```

### 运行特定测试
```bash
# 运行健康检查测试
go test -run TestHealthEndpoint ./test/

# 运行持有者端点测试
go test -run "TestHoldersEndpoint.*" ./test/

# 运行排序功能测试
go test -run "TestHoldersEndpoint.*Sort" ./test/

# 运行过滤功能测试
go test -run "TestHoldersEndpoint.*Filter" ./test/



# 运行所有 API 相关测试
go test -run "Test.*Endpoint" ./test/
```

### 详细输出
```bash
# 显示详细测试输出
go test -v ./test/

# 显示测试覆盖率
go test -cover ./test/
```

## 测试开发指南

### 测试命名规范
- 测试函数以 `Test` 开头
- 使用驼峰命名法
- 函数名应清楚描述测试内容

### 测试文件组织
- `api_test.go` - HTTP API 端点测试
- `unit_test.go` - 单元测试（函数级别）
- `integration_test.go` - 集成测试（系统级别）

### 测试数据
建议在 `test/` 目录下创建 `testdata/` 子目录存放测试数据文件。

## 待完成的测试

- [ ] 完善 API 端点测试实现
- [ ] 添加数据库操作单元测试
- [ ] 添加 Solana RPC 调用测试
- [ ] 添加集成测试
- [ ] 添加性能测试
- [ ] 添加并发测试

## 测试环境

测试建议使用独立的测试数据库，避免影响开发和生产数据：

```bash
# 创建测试数据库
CREATE DATABASE solana_spl_holder_test;

# 设置测试环境变量
export TEST_DB_CONN="root:123456@tcp(localhost:3306)/solana_spl_holder_test?charset=utf8mb4&parseTime=True&loc=Local"
```

## 详细测试用例列表

### 健康检查测试
- `TestHealthEndpoint` - 基础健康检查
- `TestHealthEndpointMethodNotAllowed` - 不支持的 HTTP 方法测试

### 持有者端点测试

#### 基础功能测试
- `TestHoldersEndpoint` - 基础查询功能
- `TestHoldersEndpointPagination` - 分页功能测试
- `TestHoldersEndpointFiltering` - mint_address 过滤测试
- `TestHoldersEndpointAmountFiltering` - 金额过滤测试

#### 排序功能测试
- `TestHoldersEndpointSorting` - 升序排序测试
  - 测试 `sort=ui_amount` 按金额升序
  - 测试 `sort=pubkey` 按公钥升序
- `TestHoldersEndpointSortingDescending` - 降序排序测试
  - 测试 `sort=-ui_amount` 按金额降序
  - 测试 `sort=-pubkey` 按公钥降序

#### 状态过滤测试
- `TestHoldersEndpointStateFiltering` - 状态过滤测试
  - 测试 `state=initialized` 过滤
  - 测试 `state=frozen` 过滤
  - 测试无效状态处理

#### 组合功能测试
- `TestHoldersEndpointCombinedSortingAndFiltering` - 排序和状态过滤组合测试
- `TestHoldersEndpointMintAddressAndStateFiltering` - mint_address 和 state 双重过滤测试
  - 测试匹配条件的查询
  - 测试不匹配条件的查询
  - 测试不存在数据的查询

### SPL Token 端点测试
- `TestSPLEndpointGet` - 获取所有 SPL Token
- `TestSPLEndpointCreate` - 创建新 SPL Token
- `TestSPLEndpointCreateInvalid` - 创建无效 SPL Token 测试
- `TestSPLEndpointGetByMintAddress` - 根据 mint_address 获取
- `TestSPLEndpointUpdateByMintAddress` - 根据 mint_address 更新
- `TestSPLEndpointDeleteByMintAddress` - 根据 mint_address 删除

### 系统测试
- `TestAPIDocumentation` - API 文档端点测试
- `TestMethodNotAllowed` - 不支持的 HTTP 方法测试
- `TestDatabaseIntegration` - 数据库集成测试
- `TestConcurrentRequests` - 并发请求测试

### 性能测试
- `BenchmarkHealthEndpoint` - 健康检查性能测试
- `BenchmarkHoldersEndpoint` - 持有者查询性能测试

## 测试覆盖的功能

✅ **已实现的测试功能：**
- HTTP 端点基础功能
- 分页查询
- 数据过滤（mint_address、state）
- 数据排序（ui_amount、pubkey，升序/降序）
- 参数组合使用
- 错误处理和边界条件
- 并发安全性
- 性能基准测试

🔄 **测试数据模拟：**
- 5 条模拟 Holder 数据
- 包含不同状态（initialized、frozen）
- 包含不同金额（0 到 3.0）
- 覆盖各种查询场景