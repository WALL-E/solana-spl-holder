#!/bin/bash

# 测试 /holders 接口的 state 查询参数功能

echo "🧪 测试 Holders API 的 state 查询参数功能"
echo "======================================"

BASE_URL="http://localhost:8091"

# 测试 1: 查询 frozen 状态的 holders
echo "\n📋 测试 1: 查询 state=frozen 的 holders"
response=$(curl -s "$BASE_URL/holders?state=frozen")
status=$(echo $response | jq -r '.success')
count=$(echo $response | jq -r '.total')
if [ "$status" = "true" ]; then
    echo "✅ 成功: 找到 $count 个 Frozen 状态的 holders"
else
    echo "❌ 失败: $(echo $response | jq -r '.error')"
fi

# 测试 2: 查询 initialized 状态的 holders
echo "\n📋 测试 2: 查询 state=initialized 的 holders"
response=$(curl -s "$BASE_URL/holders?state=initialized")
status=$(echo $response | jq -r '.success')
count=$(echo $response | jq -r '.total')
if [ "$status" = "true" ]; then
    echo "✅ 成功: 找到 $count 个 Initialized 状态的 holders"
else
    echo "❌ 失败: $(echo $response | jq -r '.error')"
fi

# 测试 3: 查询 initialized 状态的 holders (小写)
echo "\n📋 测试 3: 查询 state=initialized 的 holders (小写)"
response=$(curl -s "$BASE_URL/holders?state=initialized")
status=$(echo $response | jq -r '.success')
count=$(echo $response | jq -r '.total')
if [ "$status" = "true" ]; then
    echo "✅ 成功: 找到 $count 个 initialized 状态的 holders"
else
    echo "❌ 失败: $(echo $response | jq -r '.error')"
fi

# 测试 4: 组合查询 - state + mint_address
echo "\n📋 测试 4: 组合查询 state=frozen&mint_address=Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg"
response=$(curl -s "$BASE_URL/holders?state=frozen&mint_address=Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg")
status=$(echo $response | jq -r '.success')
count=$(echo $response | jq -r '.total')
if [ "$status" = "true" ]; then
    echo "✅ 成功: 找到 $count 个符合条件的 holders"
else
    echo "❌ 失败: $(echo $response | jq -r '.error')"
fi

# 测试 5: 查询不存在的状态
echo "\n📋 测试 5: 查询不存在的状态 state=NonExistent"
response=$(curl -s "$BASE_URL/holders?state=NonExistent")
status=$(echo $response | jq -r '.success')
count=$(echo $response | jq -r '.total')
if [ "$status" = "true" ]; then
    echo "✅ 成功: 找到 $count 个 NonExistent 状态的 holders (应该为0)"
else
    echo "❌ 失败: $(echo $response | jq -r '.error')"
fi

# 测试 6: 验证 API 文档包含 state 参数
echo "\n📋 测试 6: 验证 API 文档包含 state 参数说明"
response=$(curl -s "$BASE_URL/")
if echo "$response" | grep -q "state.*string.*按状态筛选"; then
    echo "✅ 成功: API 文档包含 state 参数说明"
else
    echo "❌ 失败: API 文档未包含 state 参数说明"
fi

echo "\n🎉 state 查询参数功能测试完成！"