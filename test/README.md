# Test - 测试目录

本目录包含项目的测试文件和测试工具。

## 文件说明

### api_test.go
API 端点测试文件，包含：
- 健康检查端点测试
- 持有者查询端点测试
- SPL Token 端点测试
- API 文档端点测试

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
# 运行 API 测试
go test -run TestHealthEndpoint ./test/

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