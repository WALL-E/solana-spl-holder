#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

$SCRIPT_DIR/solana-spl-holder \
    --rpc_url https://api.devnet.solana.com \
    --interval_time 30
