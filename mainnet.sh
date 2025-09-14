#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

# 从环境变量中获取SOLANA_RPC
if [ -z "$SOLANA_RPC" ]; then
    echo "错误：SOLANA_RPC 环境变量未设置"
    exit 1
fi

# 进入server目录运行服务
cd "$SCRIPT_DIR/server"
go run main.go \
    --rpc_url $SOLANA_RPC \
    --interval_time 300
