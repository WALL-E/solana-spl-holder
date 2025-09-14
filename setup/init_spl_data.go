package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// SPL Token 数据结构
type SPLToken struct {
	Symbol      string
	MintAddress string
}

// 预定义的 SPL Token 列表
var defaultSPLTokens = []SPLToken{
	{"TSLAx", "XsDoVfqeBukxuZHWhdvWHBhgEHjGNst4MLodqsJHzoB"},
	{"AAPLx", "XsbEhLAtcf6HdfpFZ5xEMdqW8nfAvcsP5bdudRLJzJp"},
	{"NVDAx", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh"},
	{"AMZNx", "Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg"},
	{"COINx", "Xs7ZdzSHLU9ftNJsii5fCeJhoRWSC32SQGzGQtePxNu"},
	{"HOODx", "XsvNBAYkrDRNhA7wPHQfX3ZUXZyZLdnCQDfHZ56bzpg"},
	{"GOOGLx", "XsCPL9dNWBMvFtTmwcCA5v3xWPSMEBCszbQdiLLq6aN"},
}

func main() {
	// 数据库连接字符串
	dbConn := "root:123456@tcp(localhost:3306)/solana_spl_holder?charset=utf8mb4&parseTime=True&loc=Local"

	// 连接数据库
	db, err := sql.Open("mysql", dbConn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	fmt.Println("数据库连接成功")

	// 初始化 SPL Token 数据
	if err := initSPLTokens(db); err != nil {
		log.Fatalf("初始化 SPL Token 数据失败: %v", err)
	}

	fmt.Println("SPL Token 数据初始化完成")
}

// 初始化 SPL Token 数据
func initSPLTokens(db *sql.DB) error {
	// 准备插入语句
	stmt, err := db.Prepare(`
		INSERT INTO spl (symbol, mint_address) 
		VALUES (?, ?) 
		ON DUPLICATE KEY UPDATE 
			symbol = VALUES(symbol),
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("准备插入语句失败: %v", err)
	}
	defer stmt.Close()

	// 插入每个 SPL Token
	for _, token := range defaultSPLTokens {
		_, err := stmt.Exec(token.Symbol, token.MintAddress)
		if err != nil {
			return fmt.Errorf("插入 SPL Token %s 失败: %v", token.Symbol, err)
		}
		fmt.Printf("✓ 插入/更新 SPL Token: %s (%s)\n", token.Symbol, token.MintAddress)
	}

	return nil
}
