#!/bin/bash

go mod tidy
go build -ldflags="-s -w" -o solana-spl-holder main.go
