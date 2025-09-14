#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

# 进入server目录运行服务
cd "$SCRIPT_DIR/server"
go run main.go \
    --rpc_url https://api.devnet.solana.com \
    --interval_time 30
