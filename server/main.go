package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

// 全局日志记录器
var (
	logger   = log.New(os.Stdout, "[solana-spl-holder] ", log.LstdFlags|log.Lshortfile)
	errorLog = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile)
	infoLog  = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	debugLog = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
)

// 获取当前Go协程ID（仅用于日志调试）
func getGoroutineID() string {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	stack := string(buf[:n])
	var id string
	fmt.Sscanf(stack, "goroutine %s ", &id)
	return id
}

// 错误包装函数
func wrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// 日志辅助函数
func logError(operation string, err error) {
	if err != nil {
		errorLog.Printf("%s: %v", operation, err)
	}
}

func logInfo(format string, args ...interface{}) {
	infoLog.Printf(format, args...)
}

func logDebug(format string, args ...interface{}) {
	debugLog.Printf(format, args...)
}

// =================================================================
// 1. 数据结构定义 (用于JSON解析和数据库映射)
// =================================================================

// RPCRequest 定义了发送到 Solana RPC 的请求体结构
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// RPCResponse 定义了从 Solana RPC 返回的响应体结构
type RPCResponse struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      string       `json:"id"`
	Result  []ResultItem `json:"result"`
	Error   *RPCError    `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ResultItem 对应响应中 result 数组的每个元素
type ResultItem struct {
	Pubkey  string  `json:"pubkey"`
	Account Account `json:"account"`
}

// Account 对应账户信息
type Account struct {
	Lamports uint64 `json:"lamports"`
	Data     Data   `json:"data"`
	Owner    string `json:"owner"`
}

// Data 对应账户数据
type Data struct {
	Parsed Parsed `json:"parsed"`
}

// Parsed 对应解析后的数据
type Parsed struct {
	Info Info   `json:"info"`
	Type string `json:"type"`
}

// Info 包含详细的代币信息
type Info struct {
	IsNative    bool        `json:"isNative"`
	Owner       string      `json:"owner"`
	State       string      `json:"state"`
	TokenAmount TokenAmount `json:"tokenAmount"`
}

// TokenAmount 包含代币数量信息
type TokenAmount struct {
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	UIAmount       float64 `json:"uiAmount"`
	UIAmountString string  `json:"uiAmountString"`
}

// Holder 对应数据库中的 'holder' 表结构
type Holder struct {
	ID             int64     `json:"id"`
	MintAddress    string    `json:"mint_address"`
	Pubkey         string    `json:"pubkey"`
	Lamports       uint64    `json:"lamports"`
	IsNative       bool      `json:"isNative"`
	Owner          string    `json:"owner"`
	State          string    `json:"state"`
	Decimals       int       `json:"decimals"`
	Amount         string    `json:"amount"`
	UIAmount       float64   `json:"uiAmount"`
	UIAmountString string    `json:"uiAmountString"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// SPL 对应数据库中的 'spl' 表结构
type SPL struct {
	ID          int       `json:"id"`
	Symbol      string    `json:"symbol"`
	MintAddress string    `json:"mint_address"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SPLCreateRequest 创建SPL的请求结构
type SPLCreateRequest struct {
	Symbol      string `json:"symbol" validate:"required,min=1,max=255"`
	MintAddress string `json:"mint_address" validate:"required,min=32,max=255"`
}

// SPLUpdateRequest 更新SPL的请求结构
type SPLUpdateRequest struct {
	Symbol      string `json:"symbol" validate:"required,min=1,max=255"`
	MintAddress string `json:"mint_address" validate:"required,min=32,max=255"`
}

// HolderUpdateRequest 更新Holder状态的请求结构
type HolderUpdateRequest struct {
	State string `json:"state" validate:"required"`
}

// 验证SPL创建请求
func (req *SPLCreateRequest) Validate() error {
	if strings.TrimSpace(req.Symbol) == "" {
		return fmt.Errorf("symbol不能为空")
	}
	if len(req.Symbol) > 255 {
		return fmt.Errorf("symbol长度不能超过255个字符")
	}
	if strings.TrimSpace(req.MintAddress) == "" {
		return fmt.Errorf("mint_address不能为空")
	}
	if len(req.MintAddress) < 32 || len(req.MintAddress) > 255 {
		return fmt.Errorf("mint_address长度必须在32-255个字符之间")
	}
	return nil
}

// 验证SPL更新请求
func (req *SPLUpdateRequest) Validate() error {
	if strings.TrimSpace(req.Symbol) == "" {
		return fmt.Errorf("symbol不能为空")
	}
	if len(req.Symbol) > 255 {
		return fmt.Errorf("symbol长度不能超过255个字符")
	}
	if strings.TrimSpace(req.MintAddress) == "" {
		return fmt.Errorf("mint_address不能为空")
	}
	if len(req.MintAddress) < 32 || len(req.MintAddress) > 255 {
		return fmt.Errorf("mint_address长度必须在32-255个字符之间")
	}
	return nil
}

// 验证Holder更新请求
func (req *HolderUpdateRequest) Validate() error {
	validStates := []string{"Uninitialized", "Initialized", "Frozen"}
	for _, validState := range validStates {
		if req.State == validState {
			return nil
		}
	}
	return fmt.Errorf("state必须是以下值之一: %v", validStates)
}

// 查询spl表所有mint_address
func getAllMintAddresses(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT mint_address FROM spl")
	if err != nil {
		return nil, wrapError("查询mint_address列表", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var mint string
		if err := rows.Scan(&mint); err != nil {
			return nil, wrapError("扫描mint_address", err)
		}
		result = append(result, mint)
	}

	if err := rows.Err(); err != nil {
		return nil, wrapError("遍历查询结果", err)
	}

	logInfo("成功获取到 %d 个mint地址", len(result))
	return result, nil
}

// MariaDB插入/更新
func upsertHolderMariaDB(dbOrTx interface{}, mintAddress string, item ResultItem) error {
	// 数据验证
	if mintAddress == "" {
		return fmt.Errorf("mint地址不能为空")
	}
	if item.Pubkey == "" {
		return fmt.Errorf("pubkey不能为空")
	}

	info := item.Account.Data.Parsed.Info
	sqlStr := `INSERT INTO holder (
		mint_address, pubkey, lamports, is_native, owner, state, decimals, amount, ui_amount, ui_amount_string, created_at, updated_at
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
	) ON DUPLICATE KEY UPDATE
		lamports = VALUES(lamports),
		is_native = VALUES(is_native),
		owner = VALUES(owner),
		state = VALUES(state),
		decimals = VALUES(decimals),
		amount = VALUES(amount),
		ui_amount = VALUES(ui_amount),
		ui_amount_string = VALUES(ui_amount_string),
		updated_at = CURRENT_TIMESTAMP;`

	var execFn func(string, ...interface{}) (sql.Result, error)
	switch v := dbOrTx.(type) {
	case *sql.DB:
		execFn = v.Exec
	case *sql.Tx:
		execFn = v.Exec
	default:
		return fmt.Errorf("无效的数据库连接类型")
	}

	_, err := execFn(sqlStr,
		mintAddress,
		item.Pubkey,
		item.Account.Lamports,
		info.IsNative,
		info.Owner,
		info.State,
		info.TokenAmount.Decimals,
		info.TokenAmount.Amount,
		info.TokenAmount.UIAmount,
		info.TokenAmount.UIAmountString,
	)
	if err != nil {
		return wrapError(fmt.Sprintf("更新持有者数据(pubkey: %s)", item.Pubkey), err)
	}
	return nil
}

// 检查表是否存在
func checkTableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count)
	if err != nil {
		return false, wrapError(fmt.Sprintf("检查表%s是否存在", tableName), err)
	}
	return count > 0, nil
}

// MariaDB初始化
func initMariaDB(connStr string) (*sql.DB, error) {
	if connStr == "" {
		return nil, fmt.Errorf("数据库连接字符串不能为空")
	}

	logInfo("正在连接数据库...")
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, wrapError("打开数据库连接", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, wrapError("数据库连接测试", err)
	}

	logInfo("数据库连接成功")
	logInfo("正在检查数据库表结构...")

	// 检查holder表是否存在
	holderExists, err := checkTableExists(db, "holder")
	if err != nil {
		return nil, err
	}

	if holderExists {
		logInfo("holder表已存在，跳过创建")
	} else {
		logInfo("holder表不存在，正在创建...")
		createHolderTable := `CREATE TABLE holder (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			mint_address VARCHAR(255) NOT NULL,
			pubkey VARCHAR(255) NOT NULL,
			lamports BIGINT NOT NULL,
			is_native TINYINT(1) NOT NULL,
			owner VARCHAR(255) NOT NULL,
			state VARCHAR(50) NOT NULL,
			decimals INT NOT NULL,
			amount DECIMAL(38,0) NOT NULL,
			ui_amount DECIMAL(38,6) NOT NULL,
			ui_amount_string VARCHAR(255) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_holder_mint_pubkey (mint_address, pubkey),
			INDEX idx_mint_address (mint_address),
			INDEX idx_pubkey (pubkey),
			INDEX idx_owner (owner),
			INDEX idx_updated_at (updated_at)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;`
		_, err = db.Exec(createHolderTable)
		if err != nil {
			return nil, wrapError("创建holder表", err)
		}
		logInfo("holder表创建成功")
	}

	// 检查spl表是否存在
	splExists, err := checkTableExists(db, "spl")
	if err != nil {
		return nil, err
	}

	if splExists {
		logInfo("spl表已存在，跳过创建")
	} else {
		logInfo("spl表不存在，正在创建...")
		createSplTable := `CREATE TABLE IF NOT EXISTS spl (
		id INT AUTO_INCREMENT PRIMARY KEY,
		symbol VARCHAR(255) NOT NULL,
		mint_address VARCHAR(255) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY unique_mint_address (mint_address)
	) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;`
		_, err = db.Exec(createSplTable)
		if err != nil {
			return nil, wrapError("创建spl表", err)
		}
		logInfo("spl表创建成功")
	}

	logInfo("数据库表结构检查完成")
	return db, nil
}

// API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Total   int         `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
	Limit   int         `json:"limit,omitempty"`
}

// 发送JSON响应
func sendJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// 创建SPL记录
func createSPL(db *sql.DB, req *SPLCreateRequest) (*SPL, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, wrapError("validation failed", err)
	}

	// 检查mint_address是否已存在
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM spl WHERE mint_address = ?", req.MintAddress).Scan(&count)
	if err != nil {
		return nil, wrapError("failed to check existing mint_address", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("mint_address已存在: %s", req.MintAddress)
	}

	// 插入新记录
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO spl (symbol, mint_address, created_at, updated_at) VALUES (?, ?, ?, ?)",
		req.Symbol, req.MintAddress, now, now,
	)
	if err != nil {
		return nil, wrapError("failed to insert SPL record", err)
	}

	// 获取插入的ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, wrapError("failed to get last insert ID", err)
	}

	// 返回创建的记录
	spl := &SPL{
		ID:          int(id),
		Symbol:      req.Symbol,
		MintAddress: req.MintAddress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	logInfo("Created SPL record: ID=%d, Symbol=%s, MintAddress=%s", id, req.Symbol, req.MintAddress)
	return spl, nil
}

// 处理创建SPL的HTTP请求
func handleCreateSPL(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
			return
		}

		// 解析请求体
		var req SPLCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logError("Failed to decode request body", err)
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid JSON format",
			})
			return
		}

		// 创建SPL记录
		spl, err := createSPL(db, &req)
		if err != nil {
			logError("Failed to create SPL", err)
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// 返回成功响应
		sendJSONResponse(w, http.StatusCreated, APIResponse{
			Success: true,
			Data:    spl,
		})
	}
}

// 获取SPL记录列表（支持分页）
func getSPLList(db *sql.DB, page, limit int) ([]SPL, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 10
	}

	// 计算偏移量
	offset := (page - 1) * limit

	// 获取总数
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM spl").Scan(&total)
	if err != nil {
		return nil, 0, wrapError("failed to get total count", err)
	}

	// 查询数据
	rows, err := db.Query(
		"SELECT id, symbol, mint_address, created_at, updated_at FROM spl ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, 0, wrapError("failed to query SPL records", err)
	}
	defer rows.Close()

	var spls []SPL
	for rows.Next() {
		var spl SPL
		err := rows.Scan(&spl.ID, &spl.Symbol, &spl.MintAddress, &spl.CreatedAt, &spl.UpdatedAt)
		if err != nil {
			return nil, 0, wrapError("failed to scan SPL record", err)
		}
		spls = append(spls, spl)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, wrapError("error iterating SPL records", err)
	}

	return spls, total, nil
}

// 根据ID获取SPL记录
func getSPLByID(db *sql.DB, id int) (*SPL, error) {
	var spl SPL
	err := db.QueryRow(
		"SELECT id, symbol, mint_address, created_at, updated_at FROM spl WHERE id = ?",
		id,
	).Scan(&spl.ID, &spl.Symbol, &spl.MintAddress, &spl.CreatedAt, &spl.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("SPL记录不存在: ID=%d", id)
		}
		return nil, wrapError("failed to get SPL by ID", err)
	}

	return &spl, nil
}

// 根据mint_address获取SPL记录
func getSPLByMintAddress(db *sql.DB, mintAddress string) (*SPL, error) {
	var spl SPL
	err := db.QueryRow(
		"SELECT id, symbol, mint_address, created_at, updated_at FROM spl WHERE mint_address = ?",
		mintAddress,
	).Scan(&spl.ID, &spl.Symbol, &spl.MintAddress, &spl.CreatedAt, &spl.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("SPL记录不存在: mint_address=%s", mintAddress)
		}
		return nil, wrapError("failed to get SPL by mint_address", err)
	}

	return &spl, nil
}

// 处理获取SPL列表的HTTP请求
func handleGetSPLList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
			return
		}

		// 解析查询参数
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		page := 1
		limit := 10

		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}

		// 获取SPL列表
		spls, total, err := getSPLList(db, page, limit)
		if err != nil {
			logError("Failed to get SPL list", err)
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "Failed to get SPL list",
			})
			return
		}

		// 计算分页信息
		totalPages := (total + limit - 1) / limit
		hasNext := page < totalPages
		hasPrev := page > 1

		// 返回响应
		response := map[string]interface{}{
			"data": spls,
			"pagination": map[string]interface{}{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
				"has_next":    hasNext,
				"has_prev":    hasPrev,
			},
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    response,
		})
	}
}

// 处理根据mint_address获取SPL的HTTP请求
func handleGetSPLByMintAddress(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
			return
		}

		// 从URL路径中提取mint_address
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Missing SPL mint_address",
			})
			return
		}

		mintAddress := parts[len(parts)-1]
		if mintAddress == "" {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid SPL mint_address",
			})
			return
		}

		// 获取SPL记录
		spl, err := getSPLByMintAddress(db, mintAddress)
		if err != nil {
			logError("Failed to get SPL by mint_address", err)
			if strings.Contains(err.Error(), "不存在") {
				sendJSONResponse(w, http.StatusNotFound, APIResponse{
					Success: false,
					Error:   err.Error(),
				})
			} else {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Error:   "Failed to get SPL record",
				})
			}
			return
		}

		// 返回成功响应
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    spl,
		})
	}
}

// 更新SPL记录
func updateSPL(db *sql.DB, id int, req *SPLUpdateRequest) (*SPL, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, wrapError("validation failed", err)
	}

	// 检查记录是否存在
	existingSPL, err := getSPLByID(db, id)
	if err != nil {
		return nil, err
	}

	// 检查mint_address是否被其他记录使用
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM spl WHERE mint_address = ? AND id != ?", req.MintAddress, id).Scan(&count)
	if err != nil {
		return nil, wrapError("failed to check existing mint_address", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("mint_address已被其他记录使用: %s", req.MintAddress)
	}

	// 更新记录
	now := time.Now()
	_, err = db.Exec(
		"UPDATE spl SET symbol = ?, mint_address = ?, updated_at = ? WHERE id = ?",
		req.Symbol, req.MintAddress, now, id,
	)
	if err != nil {
		return nil, wrapError("failed to update SPL record", err)
	}

	// 返回更新后的记录
	updatedSPL := &SPL{
		ID:          existingSPL.ID,
		Symbol:      req.Symbol,
		MintAddress: req.MintAddress,
		CreatedAt:   existingSPL.CreatedAt,
		UpdatedAt:   now,
	}

	logInfo("Updated SPL record: ID=%d, Symbol=%s, MintAddress=%s", id, req.Symbol, req.MintAddress)
	return updatedSPL, nil
}

// 根据mint_address更新SPL记录
func updateSPLByMintAddress(db *sql.DB, mintAddress string, req *SPLUpdateRequest) (*SPL, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, wrapError("validation failed", err)
	}

	// 检查记录是否存在
	existingSPL, err := getSPLByMintAddress(db, mintAddress)
	if err != nil {
		return nil, err
	}

	// 检查新的mint_address是否被其他记录使用
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM spl WHERE mint_address = ? AND mint_address != ?", req.MintAddress, mintAddress).Scan(&count)
	if err != nil {
		return nil, wrapError("failed to check existing mint_address", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("mint_address已被其他记录使用: %s", req.MintAddress)
	}

	// 更新记录
	now := time.Now()
	_, err = db.Exec(
		"UPDATE spl SET symbol = ?, mint_address = ?, updated_at = ? WHERE mint_address = ?",
		req.Symbol, req.MintAddress, now, mintAddress,
	)
	if err != nil {
		return nil, wrapError("failed to update SPL record", err)
	}

	// 返回更新后的记录
	updatedSPL := &SPL{
		ID:          existingSPL.ID,
		Symbol:      req.Symbol,
		MintAddress: req.MintAddress,
		CreatedAt:   existingSPL.CreatedAt,
		UpdatedAt:   now,
	}

	logInfo("Updated SPL record: MintAddress=%s, Symbol=%s, NewMintAddress=%s", mintAddress, req.Symbol, req.MintAddress)
	return updatedSPL, nil
}

// 处理更新SPL的HTTP请求
func handleUpdateSPL(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
			return
		}

		// 从URL路径中提取mint_address
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Missing SPL mint_address",
			})
			return
		}

		mintAddress := parts[len(parts)-1]
		if mintAddress == "" {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid SPL mint_address",
			})
			return
		}

		// 解析请求体
		var req SPLUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logError("Failed to decode request body", err)
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid JSON format",
			})
			return
		}

		// 更新SPL记录
		spl, err := updateSPLByMintAddress(db, mintAddress, &req)
		if err != nil {
			logError("Failed to update SPL", err)
			if strings.Contains(err.Error(), "不存在") {
				sendJSONResponse(w, http.StatusNotFound, APIResponse{
					Success: false,
					Error:   err.Error(),
				})
			} else {
				sendJSONResponse(w, http.StatusBadRequest, APIResponse{
					Success: false,
					Error:   err.Error(),
				})
			}
			return
		}

		// 返回成功响应
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    spl,
		})
	}
}

// 删除SPL记录
func deleteSPL(db *sql.DB, id int) error {
	// 检查记录是否存在
	_, err := getSPLByID(db, id)
	if err != nil {
		return err
	}

	// 删除记录
	result, err := db.Exec("DELETE FROM spl WHERE id = ?", id)
	if err != nil {
		return wrapError("failed to delete SPL record", err)
	}

	// 检查是否真的删除了记录
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return wrapError("failed to get rows affected", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("SPL记录删除失败: ID=%d", id)
	}

	logInfo("Deleted SPL record: ID=%d", id)
	return nil
}

// 根据mint_address删除SPL记录
func deleteSPLByMintAddress(db *sql.DB, mintAddress string) error {
	// 检查记录是否存在
	_, err := getSPLByMintAddress(db, mintAddress)
	if err != nil {
		return err
	}

	// 删除记录
	result, err := db.Exec("DELETE FROM spl WHERE mint_address = ?", mintAddress)
	if err != nil {
		return wrapError("failed to delete SPL record", err)
	}

	// 检查是否真的删除了记录
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return wrapError("failed to get rows affected", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("SPL记录删除失败: mint_address=%s", mintAddress)
	}

	logInfo("Deleted SPL record: mint_address=%s", mintAddress)
	return nil
}

// 更新Holder状态
func updateHolderState(db *sql.DB, mintAddress, pubkey, state string) (*Holder, error) {
	// 检查记录是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM holder WHERE mint_address = ? AND pubkey = ?)", mintAddress, pubkey).Scan(&exists)
	if err != nil {
		return nil, wrapError("检查Holder记录是否存在", err)
	}

	if !exists {
		return nil, fmt.Errorf("mint_address为 %s 且 pubkey为 %s 的Holder记录不存在", mintAddress, pubkey)
	}

	// 更新状态
	_, err = db.Exec("UPDATE holder SET state = ?, updated_at = CURRENT_TIMESTAMP WHERE mint_address = ? AND pubkey = ?", state, mintAddress, pubkey)
	if err != nil {
		return nil, wrapError("更新Holder状态", err)
	}

	// 查询更新后的记录
	var holder Holder
	err = db.QueryRow(`
		SELECT id, mint_address, pubkey, lamports, is_native, owner, state, decimals, 
		       amount, ui_amount, ui_amount_string, created_at, updated_at 
		FROM holder 
		WHERE mint_address = ? AND pubkey = ?
	`, mintAddress, pubkey).Scan(
		&holder.ID, &holder.MintAddress, &holder.Pubkey, &holder.Lamports,
		&holder.IsNative, &holder.Owner, &holder.State, &holder.Decimals,
		&holder.Amount, &holder.UIAmount, &holder.UIAmountString,
		&holder.CreatedAt, &holder.UpdatedAt,
	)
	if err != nil {
		return nil, wrapError("查询更新后的Holder记录", err)
	}

	return &holder, nil
}

// 处理更新Holder状态的HTTP请求
func handleUpdateHolderState(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
			return
		}

		// 从URL路径中提取mint_address和pubkey
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 3 {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Missing mint_address or pubkey",
			})
			return
		}

		mintAddress := parts[len(parts)-2]
		pubkey := parts[len(parts)-1]
		if mintAddress == "" || pubkey == "" {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid mint_address or pubkey",
			})
			return
		}

		// 解析请求体
		var req HolderUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logError("Failed to decode request body", err)
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid JSON format",
			})
			return
		}

		// 验证请求
		if err := req.Validate(); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// 更新Holder状态
		holder, err := updateHolderState(db, mintAddress, pubkey, req.State)
		if err != nil {
			logError("Failed to update holder state", err)
			if strings.Contains(err.Error(), "不存在") {
				sendJSONResponse(w, http.StatusNotFound, APIResponse{
					Success: false,
					Error:   err.Error(),
				})
			} else {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Error:   "Failed to update holder state",
				})
			}
			return
		}

		// 返回成功响应
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    holder,
		})
	}
}

// 处理删除SPL的HTTP请求
func handleDeleteSPL(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
			return
		}

		// 从URL路径中提取mint_address
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Missing SPL mint_address",
			})
			return
		}

		mintAddress := parts[len(parts)-1]
		if mintAddress == "" {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid SPL mint_address",
			})
			return
		}

		// 删除SPL记录
		err := deleteSPLByMintAddress(db, mintAddress)
		if err != nil {
			logError("Failed to delete SPL", err)
			if strings.Contains(err.Error(), "不存在") {
				sendJSONResponse(w, http.StatusNotFound, APIResponse{
					Success: false,
					Error:   err.Error(),
				})
			} else {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Error:   "Failed to delete SPL record",
				})
			}
			return
		}

		// 返回成功响应
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"message": "SPL record deleted successfully"},
		})
	}
}

// MariaDB API处理
func apiHandlerMariaDB(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "只支持GET方法",
			})
			return
		}
		query := r.URL.Query()
		page, _ := strconv.Atoi(query.Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(query.Get("limit"))
		if limit <= 0 {
			limit = 10
		}
		if limit > 1000 {
			limit = 1000 // 限制最大查询数量
		}
		offset := (page - 1) * limit
		baseQuery := "SELECT id, mint_address, pubkey, lamports, is_native, owner, state, decimals, amount, ui_amount, ui_amount_string, created_at, updated_at FROM holder"
		var args []interface{}
		var conds []string
		// 添加amount > 0的过滤条件
		conds = append(conds, "amount > 0")
		if owner := query.Get("owner"); owner != "" {
			conds = append(conds, "owner = ?")
			args = append(args, owner)
		}
		if mint := query.Get("mint_address"); mint != "" {
			conds = append(conds, "mint_address = ?")
			args = append(args, mint)
		}
		if len(conds) > 0 {
			baseQuery += " WHERE " + strings.Join(conds, " AND ")
		}
		sort := query.Get("sort")
		if sort != "" {
			dir := "ASC"
			col := sort
			if strings.HasPrefix(sort, "-") {
				dir = "DESC"
				col = sort[1:]
			}
			baseQuery += " ORDER BY " + col + " " + dir
		}
		baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
		// 获取总数
		countQuery := "SELECT COUNT(*) FROM holder"
		if len(conds) > 0 {
			countQuery += " WHERE " + strings.Join(conds, " AND ")
		}
		var total int
		err := db.QueryRow(countQuery, args...).Scan(&total)
		if err != nil {
			logError("查询总数", err)
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "查询总数失败",
			})
			return
		}

		rows, err := db.Query(baseQuery, args...)
		if err != nil {
			logError("查询持有者数据", err)
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "查询数据失败",
			})
			return
		}
		defer rows.Close()

		var holders []Holder
		for rows.Next() {
			var h Holder
			err := rows.Scan(&h.ID, &h.MintAddress, &h.Pubkey, &h.Lamports, &h.IsNative, &h.Owner, &h.State, &h.Decimals, &h.Amount, &h.UIAmount, &h.UIAmountString, &h.CreatedAt, &h.UpdatedAt)
			if err != nil {
				logError("扫描数据行", err)
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Error:   "数据解析失败",
				})
				return
			}
			holders = append(holders, h)
		}

		if err := rows.Err(); err != nil {
			logError("遍历查询结果", err)
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "数据遍历失败",
			})
			return
		}

		if holders == nil {
			holders = []Holder{}
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    holders,
			Total:   total,
			Page:    page,
			Limit:   limit,
		})
	}
}

// fetchAndStoreData 从 RPC 获取数据并存入数据库
func fetchAndStoreData(ctx context.Context, rpcURL string, db *sql.DB, httpClient *http.Client, mintAddress string) {
	if mintAddress == "" {
		logError("获取数据", fmt.Errorf("mint地址不能为空"))
		return
	}

	requestPayload := RPCRequest{
		Jsonrpc: "2.0",
		ID:      "1",
		Method:  "getProgramAccounts",
		Params: []interface{}{
			"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb", // SPL Token Program ID
			map[string]interface{}{
				"encoding": "jsonParsed",
				"filters": []map[string]interface{}{
					{
						"memcmp": map[string]interface{}{
							"offset": 0, // `mint` 字段的偏移量是 0
							"bytes":  mintAddress,
						},
					},
				},
			},
		},
	}

	reqBodyBytes, err := json.Marshal(requestPayload)
	if err != nil {
		logError("序列化请求体", err)
		return
	}

	logInfo("开始获取 SPL token 账户信息: %s", mintAddress)

	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		logError("创建HTTP请求", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "solana-spl-holder/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		logError("执行HTTP请求", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logError("HTTP请求失败", fmt.Errorf("状态码: %d, 状态: %s", resp.StatusCode, resp.Status))
		return
	}

	var rpcResponse RPCResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&rpcResponse); decodeErr != nil {
		logError("解析JSON响应", err)
		return
	}

	if rpcResponse.Error != nil {
		logError("RPC调用失败", fmt.Errorf("代码: %d, 消息: %s", rpcResponse.Error.Code, rpcResponse.Error.Message))
		return
	}

	if len(rpcResponse.Result) == 0 {
		logInfo("mint地址 %s 未发现持有者记录", mintAddress)
		return
	}

	// 使用事务批量更新
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logError("开始数据库事务", err)
		return
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			logError("回滚事务", err)
		}
	}()

	upsertedCount := 0
	skippedCount := 0
	for _, item := range rpcResponse.Result {
		if item.Account.Data.Parsed.Type != "account" {
			skippedCount++
			continue
		}
		if err := upsertHolderMariaDB(tx, mintAddress, item); err != nil {
			logError(fmt.Sprintf("更新记录(pubkey: %s)", item.Pubkey), err)
			skippedCount++
			continue // 采集失败时跳过该条
		}
		upsertedCount++
	}

	if err := tx.Commit(); err != nil {
		logError("提交数据库事务", err)
		return
	}
	logInfo("mint地址 %s: 成功处理 %d 条记录，跳过 %d 条记录", mintAddress, upsertedCount, skippedCount)
}

func worker(ctx context.Context, rpcURL string, db *sql.DB) {
	startTime := time.Now()
	logInfo("[goroutine:%s] 数据采集任务开始", getGoroutineID())

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	mintAddresses, err := getAllMintAddresses(db)
	if err != nil {
		logError("获取mint地址列表", err)
		return
	}

	if len(mintAddresses) == 0 {
		logInfo("[goroutine:%s] spl表中没有mint地址，跳过本次采集", getGoroutineID())
		return
	}

	logInfo("开始处理 %d 个mint地址", len(mintAddresses))
	successCount := 0
	for i, mintAddress := range mintAddresses {
		select {
		case <-ctx.Done():
			logInfo("收到取消信号，停止数据采集")
			return
		default:
			logDebug("处理第 %d/%d 个mint地址: %s", i+1, len(mintAddresses), mintAddress)
			fetchAndStoreData(ctx, rpcURL, db, httpClient, mintAddress)
			successCount++

			// 添加小延迟避免过于频繁的请求
			if i < len(mintAddresses)-1 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	duration := time.Since(startTime)
	logInfo("[goroutine:%s] 数据采集任务完成，处理了 %d/%d 个地址，耗时: %v", getGoroutineID(), successCount, len(mintAddresses), duration)
}

// startWorker 启动一个定时任务，周期性地获取数据
func startWorker(ctx context.Context, interval time.Duration, rpcURL string, db *sql.DB) {
	logInfo("启动定时数据采集任务，间隔: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次
	go worker(ctx, rpcURL, db)

	for {
		select {
		case <-ticker.C:
			// 在新的goroutine中执行worker，避免阻塞定时器
			go worker(ctx, rpcURL, db)
		case <-ctx.Done():
			logInfo("数据采集定时任务正在关闭")
			return
		}
	}
}

// 配置结构
type Config struct {
	RPCURL       string
	DBConnStr    string
	IntervalTime int
	ListenPort   int
}

// 验证配置
func (c *Config) Validate() error {
	if c.RPCURL == "" {
		return fmt.Errorf("RPC URL不能为空")
	}
	if c.DBConnStr == "" {
		return fmt.Errorf("数据库连接字符串不能为空")
	}
	if c.IntervalTime < 10 {
		return fmt.Errorf("采集间隔不能小于10秒")
	}
	if c.ListenPort < 1 || c.ListenPort > 65535 {
		return fmt.Errorf("监听端口必须在1-65535范围内")
	}
	return nil
}

// 生成API文档HTML
func getAPIDocumentation() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Solana SPL Holder API 文档</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        h1 { color: #333; border-bottom: 2px solid #007cba; padding-bottom: 10px; }
        h2 { color: #007cba; margin-top: 30px; }
        h3 { color: #555; }
        .endpoint { background: #f4f4f4; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .method { font-weight: bold; color: white; padding: 3px 8px; border-radius: 3px; }
        .get { background: #28a745; }
        .post { background: #007bff; }
        .put { background: #ffc107; color: black; }
        .delete { background: #dc3545; }
        .code { background: #f8f9fa; padding: 10px; border-radius: 3px; font-family: monospace; white-space: pre-wrap; }
        .response { background: #e9ecef; padding: 10px; border-radius: 3px; margin-top: 10px; white-space: pre-wrap; font-family: monospace; }
        table { border-collapse: collapse; width: 100%; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>🚀 Solana SPL Holder API 文档</h1>
    
    <h2>📋 概述</h2>
    <p>这是一个用于管理 Solana SPL Token 持有者信息的 RESTful API 服务。</p>
    <p><strong>基础URL:</strong> <code id="base-url"></code></p>
    <script>
        // 动态获取当前页面的基础URL
        document.getElementById('base-url').textContent = window.location.origin;
    </script>
    
    <h2>🔗 API 端点</h2>
    
    <h3>1. SPL Token 管理</h3>
    
    <div class="endpoint">
        <h4><span class="method post">POST</span> /spls</h4>
        <p><strong>描述:</strong> 创建新的 SPL Token 记录</p>
        <p><strong>请求体:</strong></p>
        <div class="code">{
    "symbol": "USDC",
    "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
}</div>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "id": 1,
        "symbol": "USDC",
        "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
        "created_at": "2024-01-01T12:00:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
    }
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method get">GET</span> /spls</h4>
        <p><strong>描述:</strong> 获取 SPL Token 列表（支持分页）</p>
        <p><strong>查询参数:</strong></p>
        <table>
            <tr><th>参数</th><th>类型</th><th>默认值</th><th>描述</th></tr>
            <tr><td>page</td><td>int</td><td>1</td><td>页码</td></tr>
            <tr><td>limit</td><td>int</td><td>10</td><td>每页数量（最大1000）</td></tr>
        </table>
        <p><strong>示例:</strong> <code>/spls?page=1&limit=5</code></p>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "data": [...],
        "pagination": {
            "page": 1,
            "limit": 5,
            "total": 10,
            "total_pages": 2,
            "has_next": true,
            "has_prev": false
        }
    }
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method get">GET</span> /spls/{id}</h4>
        <p><strong>描述:</strong> 根据 ID 获取单个 SPL Token 记录</p>
        <p><strong>示例:</strong> <code>/spls/1</code></p>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "id": 1,
        "symbol": "USDC",
        "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
        "created_at": "2024-01-01T12:00:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
    }
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method get">GET</span> /spls/{mint_address}</h4>
        <p><strong>描述:</strong> 根据 mint_address 获取单个 SPL Token 记录</p>
        <p><strong>示例:</strong> <code>/spls/EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v</code></p>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "id": 1,
        "symbol": "USDC",
        "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
        "created_at": "2024-01-01T12:00:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
    }
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method put">PUT</span> /spls/{id}</h4>
        <p><strong>描述:</strong> 更新指定 ID 的 SPL Token 记录</p>
        <p><strong>示例:</strong> <code>/spls/1</code></p>
        <p><strong>请求体:</strong></p>
        <div class="code">{
    "symbol": "USDC-Updated",
    "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method put">PUT</span> /spls/{mint_address}</h4>
        <p><strong>描述:</strong> 根据 mint_address 更新 SPL Token 记录</p>
        <p><strong>示例:</strong> <code>/spls/EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v</code></p>
        <p><strong>请求体:</strong></p>
        <div class="code">{
    "symbol": "USDC-Updated",
    "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
}</div>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "id": 1,
        "symbol": "USDC-Updated",
        "mint_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
        "created_at": "2024-01-01T12:00:00Z",
        "updated_at": "2024-01-01T12:05:00Z"
    }
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method delete">DELETE</span> /spls/{id}</h4>
        <p><strong>描述:</strong> 删除指定 ID 的 SPL Token 记录</p>
        <p><strong>示例:</strong> <code>/spls/1</code></p>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "message": "SPL记录删除成功"
    }
}</div>
    </div>
    
    <div class="endpoint">
        <h4><span class="method delete">DELETE</span> /spls/{mint_address}</h4>
        <p><strong>描述:</strong> 根据 mint_address 删除 SPL Token 记录</p>
        <p><strong>示例:</strong> <code>/spls/EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v</code></p>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "message": "SPL记录已删除"
    }
}</div>
    </div>
    
    <h3>2. Holder 数据查询</h3>
    
    <div class="endpoint">
        <h4><span class="method get">GET</span> /holders</h4>
        <p><strong>描述:</strong> 查询 Token 持有者信息（支持分页和筛选）</p>
        <p><strong>查询参数:</strong></p>
        <table>
            <tr><th>参数</th><th>类型</th><th>描述</th></tr>
            <tr><td>page</td><td>int</td><td>页码（默认1）</td></tr>
            <tr><td>limit</td><td>int</td><td>每页数量（默认10，最大1000）</td></tr>
            <tr><td>owner</td><td>string</td><td>按持有者地址筛选</td></tr>
            <tr><td>mint_address</td><td>string</td><td>按 mint 地址筛选</td></tr>
            <tr><td>sort</td><td>string</td><td>排序字段（加 - 前缀为降序）</td></tr>
        </table>
        <p><strong>示例:</strong> <code>/holders?page=1&limit=10&mint_address=EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v</code></p>
    </div>
    
    <div class="endpoint">
        <h4><span class="method put">PUT</span> /holders/{mint_address}/{pubkey}</h4>
        <p><strong>描述:</strong> 更新指定 Holder 的状态</p>
        <p><strong>路径参数:</strong></p>
        <table>
            <tr><th>参数</th><th>类型</th><th>描述</th></tr>
            <tr><td>mint_address</td><td>string</td><td>Token 的 mint 地址</td></tr>
            <tr><td>pubkey</td><td>string</td><td>Holder 的公钥地址</td></tr>
        </table>
        <p><strong>请求体:</strong></p>
        <div class="code">{
    "state": "Initialized"
}</div>
        <p><strong>支持的状态值:</strong></p>
        <ul>
            <li><code>Uninitialized</code> - 未初始化</li>
            <li><code>Initialized</code> - 已初始化</li>
            <li><code>Frozen</code> - 已冻结</li>
        </ul>
        <p><strong>示例:</strong> <code>/holders/Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg/13nkreFLoEtJ5rRpknHtAUgKH1yo2CychKrtVuBLmwdf</code></p>
        <p><strong>成功响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "id": 1,
        "mint_address": "Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg",
        "pubkey": "13nkreFLoEtJ5rRpknHtAUgKH1yo2CychKrtVuBLmwdf",
        "state": "Initialized",
        "owner": "13nkreFLoEtJ5rRpknHtAUgKH1yo2CychKrtVuBLmwdf",
        "amount": "1000000",
        "uiAmount": 1.0,
        "decimals": 6,
        "updatedAt": "2024-01-01T12:05:00Z"
    }
}</div>
        <p><strong>错误响应示例:</strong></p>
        <div class="response">{
    "success": false,
    "error": "state 必须是 Uninitialized、Initialized、Frozen 之一"
}</div>
    </div>

    <h3>3. 系统状态</h3>
    
    <div class="endpoint">
        <h4><span class="method get">GET</span> /health</h4>
        <p><strong>描述:</strong> 健康检查端点</p>
        <p><strong>响应示例:</strong></p>
        <div class="response">{
    "success": true,
    "data": {
        "status": "healthy",
        "version": "1.0.0"
    }
}</div>
    </div>
    
    <h2>📝 响应格式</h2>
    <p>所有 API 响应都遵循统一的 JSON 格式：</p>
    <div class="code">{
    "success": boolean,     // 请求是否成功
    "data": object,        // 响应数据（成功时）
    "error": string,       // 错误信息（失败时）
    "total": int,          // 总记录数（分页时）
    "page": int,           // 当前页码（分页时）
    "limit": int           // 每页数量（分页时）
}</div>
    
    <h2>🔧 数据验证</h2>
    <ul>
        <li><strong>symbol:</strong> 必填，长度 1-255 字符</li>
        <li><strong>mint_address:</strong> 必填，长度 32-255 字符，必须唯一</li>
    </ul>
    
    <h2>⚡ 特性</h2>
    <ul>
        <li>✅ 完整的 CRUD 操作</li>
        <li>✅ 分页支持</li>
        <li>✅ 数据验证</li>
        <li>✅ 错误处理</li>
        <li>✅ 自动数据采集</li>
        <li>✅ 健康检查</li>
    </ul>
    
    <p style="margin-top: 40px; text-align: center; color: #666;">
        <em>Solana SPL Holder API v1.0.0 - 构建于 Go</em>
    </p>
</body>
</html>`
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "solana-spl-holder",
		Short: "Solana SPL代币持有者查询工具 - 定期获取持有者数据并提供查询API",
		Long:  `这是一个用于定期获取Solana SPL代币持有者信息的工具。\n它会连接到Solana RPC节点，获取指定代币的持有者数据，\n并将数据存储到MariaDB数据库中，同时提供HTTP API供查询使用。`,
		Run:   run,
	}

	rootCmd.PersistentFlags().String("rpc_url", "https://api.devnet.solana.com", "Solana节点RPC URL")
	rootCmd.PersistentFlags().String("db_conn", "root:123456@tcp(localhost:3306)/solana_spl_holder?charset=utf8mb4&parseTime=True&loc=Local", "MariaDB连接字符串")
	rootCmd.PersistentFlags().Int("interval_time", 300, "数据采集间隔时间(秒)")
	rootCmd.PersistentFlags().Int("listen_port", 8090, "HTTP服务监听端口")

	if err := rootCmd.Execute(); err != nil {
		errorLog.Fatalf("命令执行失败: %v", err)
	}
}

func run(cmd *cobra.Command, args []string) {
	// 获取命令行参数
	rpcURL, _ := cmd.Flags().GetString("rpc_url")
	dbConnStr, _ := cmd.Flags().GetString("db_conn")
	interval, _ := cmd.Flags().GetInt("interval_time")
	port, _ := cmd.Flags().GetInt("listen_port")

	// 创建并验证配置
	config := &Config{
		RPCURL:       rpcURL,
		DBConnStr:    dbConnStr,
		IntervalTime: interval,
		ListenPort:   port,
	}

	if err := config.Validate(); err != nil {
		errorLog.Fatalf("配置验证失败: %v", err)
	}

	logInfo("=== Solana SPL 持有者查询工具启动 ===")
	logInfo("RPC URL: %s", config.RPCURL)
	logInfo("采集间隔: %d秒", config.IntervalTime)
	logInfo("监听端口: %d", config.ListenPort)

	db, err := initMariaDB(config.DBConnStr)
	if err != nil {
		errorLog.Fatalf("数据库初始化失败: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logError("关闭数据库连接", err)
		}
	}()

	// 创建带取消功能的上下文，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动后台数据采集任务
	go startWorker(ctx, time.Duration(config.IntervalTime)*time.Second, config.RPCURL, db)

	// 设置HTTP服务器
	mux := http.NewServeMux()

	// 根路径 - API文档
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(getAPIDocumentation()))
	})

	mux.HandleFunc("/holders", apiHandlerMariaDB(db))

	// Holder状态更新路由 (支持 /holders/{mint_address}/{pubkey})
	mux.HandleFunc("/holders/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handleUpdateHolderState(db)(w, r)
		default:
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
		}
	})

	// SPL CRUD API路由
	mux.HandleFunc("/spls", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateSPL(db)(w, r)
		case http.MethodGet:
			handleGetSPLList(db)(w, r)
		default:
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
		}
	})

	// SPL单个记录操作路由 (支持 /spls/{mint_address})
	mux.HandleFunc("/spls/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetSPLByMintAddress(db)(w, r)
		case http.MethodPut:
			handleUpdateSPL(db)(w, r)
		case http.MethodDelete:
			handleDeleteSPL(db)(w, r)
		default:
			sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
				Success: false,
				Error:   "Method not allowed",
			})
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"status": "healthy", "version": "1.0.0"},
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.ListenPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动HTTP服务器
	go func() {
		logInfo("HTTP服务器启动，监听端口: %d", config.ListenPort)
		logInfo("API端点: http://localhost:%d/holders", config.ListenPort)
		logInfo("健康检查: http://localhost:%d/health", config.ListenPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errorLog.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 监听中断信号以实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logInfo("=== 服务启动完成，等待信号... ===")
	<-quit // 阻塞直到接收到信号

	logInfo("收到关闭信号，开始优雅关闭...")

	// 触发worker和其他goroutine的关闭
	cancel()

	// 创建一个有超时的上下文用于关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logError("HTTP服务器关闭", err)
	} else {
		logInfo("HTTP服务器已优雅关闭")
	}

	logInfo("=== 应用已成功关闭 ===")
}
