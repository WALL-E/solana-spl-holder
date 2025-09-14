# Prompt

```

使用golang写一个程序，名称为solana-spl-status，周期性访问 Helius RPC 提供的服务，查询solana SPL代币账户信息。

请求：
curl "https://api.devnet.solana.com" \
-X POST \
-H "Content-Type: application/json" \
-d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "getProgramAccounts",
    "params": [
        "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb",
        {
            "encoding": "jsonParsed",
            "filters": [
                {
                    "memcmp": {
                        "offset": 0,
                        "bytes": "Xa"
                    }
                }
            ]
        }
    ]
}'

响应：
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": [
   {
      "pubkey": "11foMxwYYEeFRUmXZXefk3X77oTXMqYdLEjJiNjW3ba",
      "account": {
        "lamports": 2136720,
        "data": {
          "program": "spl-token-2022",
          "parsed": {
            "info": {
              "extensions": [
                {
                  "extension": "immutableOwner"
                },
                {
                  "extension": "pausableAccount"
                },
                {
                  "extension": "transferHookAccount",
                  "state": {
                    "transferring": false
                  }
                }
              ],
              "isNative": false,
              "mint": "XsDoVfqeBukxuZHWhdvWHBhgEHjGNst4MLodqsJHzoB",
              "owner": "Eh5q92xtwYp9mcczwxKsi53PQScoBdY1mmxUVd7Mtixj",
              "state": "initialized",
              "tokenAmount": {
                "amount": "100195",
                "decimals": 8,
                "uiAmount": 0.00100195,
                "uiAmountString": "0.00100195"
              }
            },
            "type": "account"
          },
          "space": 179
        },
        "owner": "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb",
        "executable": false,
        "rentEpoch": 18446744073709551615,
        "space": 179
    },
  ]
}

解析所有响应内容，存储到数据库中。

启动单独的定时任务，周期性(interval_time)访问一次 Helius RPC 服务，获取最新的solana SPL代币账户信息，并存储到数据库中。
如果数据库中已存在相同的pubkey，则更新该记录的其他字段。
请确保程序能够处理网络异常和数据库异常，并在日志中记录相关错误信息。
请确保程序能够优雅地处理 Ctrl+C 等中断信号，实现优雅关闭。

命令行使用Cobra包处理，包括以下参数的默认值：
rpc_url = "https://api.devnet.solana.com" #solana节点的RPC URL
db_name = "solana_spl_holder.db" # SQLite数据库文件名
interval_time = 300 # 每次请求间隔3600秒
listen_port = 8090 # HTTP服务监听端口


附加要求：
1. 必须检查不要import不使用的库
2. 使用Cobra来处理命令行参数
3. 所有代码合并到一个文件中
4. 每次http请求的超时时间设置为30秒，请求消息体打印到日志中
5. 必须避免github.comcom这种拼写错误
6. 优雅退出: 程序可以捕获 Ctrl+C 等中断信号，实现优雅关闭。
```