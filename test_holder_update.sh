#!/bin/bash

# 测试PUT /holders/{mint_address}/{pubkey}接口

echo "=== 测试Holder状态更新接口 ==="
echo

# 服务器地址
BASE_URL="http://localhost:8090"

# 使用实际的测试数据
MINT_ADDRESS="Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg"
PUBKEY="13nkreFLoEtJ5rRpknHtAUgKH1yo2CychKrtVuBLmwdf"

echo "1. 测试有效的state值 - Initialized"
curl -X PUT "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": "Initialized"}' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "2. 测试有效的state值 - Frozen"
curl -X PUT "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": "Frozen"}' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "3. 测试有效的state值 - Uninitialized"
curl -X PUT "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": "Uninitialized"}' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "4. 测试无效的state值"
curl -X PUT "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": "InvalidState"}' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "5. 测试空的state值"
curl -X PUT "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": ""}' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "6. 测试无效的JSON格式"
curl -X PUT "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": }' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "7. 测试不存在的holder记录"
NON_EXISTENT_MINT="11111111111111111111111111111111"
NON_EXISTENT_PUBKEY="22222222222222222222222222222222"
curl -X PUT "$BASE_URL/holders/$NON_EXISTENT_MINT/$NON_EXISTENT_PUBKEY" \
  -H "Content-Type: application/json" \
  -d '{"state": "Initialized"}' \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "8. 测试不支持的HTTP方法"
curl -X GET "$BASE_URL/holders/$MINT_ADDRESS/$PUBKEY" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "=== 测试完成 ==="