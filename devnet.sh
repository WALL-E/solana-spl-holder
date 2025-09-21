#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

# 构建
cd "$SCRIPT_DIR"
make build

# 进入build目录运行服务
cd "$SCRIPT_DIR/build"
./solana-spl-holder \
    --rpc_url https://api.devnet.solana.com \
    --listen_port 8091 \
    --interval_time 30
