#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

$SCRIPT_DIR/solana-spl-holder \
    --rpc_url http://localhost:8899 \
    --interval_time 30
