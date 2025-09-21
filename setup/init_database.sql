-- Solana SPL Holder 数据库初始化脚本
-- 创建数据库
CREATE DATABASE IF NOT EXISTS rwa
CHARACTER SET utf8mb4
COLLATE utf8mb4_general_ci;

USE rwa;

-- 创建 dummy 表
CREATE TABLE IF NOT EXISTS dummy (
    id INT AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(255) NOT NULL,
    mint_address VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_mint_address (mint_address)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- 插入初始 Dummy 数据
INSERT INTO dummy (symbol, mint_address) VALUES
('TSLAx', 'XsDoVfqeBukxuZHWhdvWHBhgEHjGNst4MLodqsJHzoB'),
('AAPLx', 'XsbEhLAtcf6HdfpFZ5xEMdqW8nfAvcsP5bdudRLJzJp'),
('NVDAx', 'Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh'),
('AMZNx', 'Xs3eBt7uRfJX8QUs4suhyU8p2M6DoUDrJyWBa8LLZsg'),
('COINx', 'Xs7ZdzSHLU9ftNJsii5fCeJhoRWSC32SQGzGQtePxNu'),
('HOODx', 'XsvNBAYkrDRNhA7wPHQfX3ZUXZyZLdnCQDfHZ56bzpg'),
('GOOGLx', 'XsCPL9dNWBMvFtTmwcCA5v3xWPSMEBCszbQdiLLq6aN');

-- 创建 SPL 视图，系统集成需要从stable_coin表中获取symbol和mint_address
DROP VIEW IF EXISTS spl;
CREATE VIEW spl AS
SELECT
  symbol ,
  mint_address
FROM dummy;

-- 创建 SPL Token Holder 表
CREATE TABLE IF NOT EXISTS holder (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    -- Token 相关
    mint_address VARCHAR(255) NOT NULL,
    
    -- 持有者信息
    pubkey VARCHAR(255) NOT NULL,
    lamports BIGINT NOT NULL,
    is_native TINYINT(1) NOT NULL,  -- 0 = false, 1 = true
    owner VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL,
    decimals INT NOT NULL,
    
    -- 金额相关
    amount DECIMAL(38,0) NOT NULL,
    ui_amount DECIMAL(38,6) NOT NULL,
    ui_amount_string VARCHAR(255) NOT NULL,
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- 索引
    UNIQUE KEY unique_holder_mint_pubkey (mint_address, pubkey),
    INDEX idx_mint_address (mint_address),
    INDEX idx_pubkey (pubkey)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;