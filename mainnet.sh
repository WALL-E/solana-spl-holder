#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

# 从环境变量中获取SOLANA_RPC
if [ -z "$SOLANA_RPC" ]; then
    echo "错误：SOLANA_RPC 环境变量未设置"
    exit 1
fi

## 构建
cd "$SCRIPT_DIR"
make build

# 进入build目录运行服务
cd "$SCRIPT_DIR/build"
./solana-spl-holder \
    --rpc_url $SOLANA_RPC \
    --interval_time 300
