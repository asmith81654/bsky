# Account Manager Service

帳號管理服務負責管理 Bluesky 帳號的 CRUD 操作、認證管理和帳號狀態監控。

## 功能特性

### 帳號管理
- 創建、讀取、更新、刪除 Bluesky 帳號
- 帳號狀態管理（活躍、非活躍、暫停、錯誤）
- 代理服務器分配和管理
- 帳號認證測試和刷新

### 認證服務
- JWT 令牌生成和驗證
- 刷新令牌管理
- 用戶登錄和登出
- 令牌黑名單機制

### 統計和監控
- 帳號統計信息
- 帳號性能指標
- 活動監控
- 代理使用情況

## API 端點

### 帳號管理
- `GET /api/v1/accounts` - 獲取帳號列表
- `POST /api/v1/accounts` - 創建新帳號
- `GET /api/v1/accounts/{id}` - 獲取特定帳號
- `PUT /api/v1/accounts/{id}` - 更新帳號
- `DELETE /api/v1/accounts/{id}` - 刪除帳號
- `POST /api/v1/accounts/{id}/test-auth` - 測試帳號認證
- `POST /api/v1/accounts/{id}/refresh-auth` - 刷新帳號認證

### 認證
- `POST /api/v1/auth/login` - 用戶登錄
- `POST /api/v1/auth/refresh` - 刷新令牌
- `POST /api/v1/auth/logout` - 用戶登出

### 統計
- `GET /api/v1/stats/accounts` - 獲取帳號統計
- `GET /api/v1/stats/accounts/{id}/metrics` - 獲取帳號指標

### 健康檢查
- `GET /health` - 服務健康檢查

## 配置

### 環境變量
- `SERVICE_PORT` - 服務端口（默認：8001）
- `DATABASE_URL` - PostgreSQL 連接字符串
- `REDIS_URL` - Redis 連接字符串
- `JWT_SECRET` - JWT 簽名密鑰
- `ENVIRONMENT` - 運行環境（development/production）

### 數據庫
服務需要連接到 PostgreSQL 數據庫，包含以下表：
- `accounts` - 帳號信息
- `proxies` - 代理服務器配置
- `tasks` - 任務記錄
- `metrics` - 性能指標

### Redis
用於：
- 刷新令牌存儲
- 令牌黑名單
- 緩存

## 開發

### 本地運行
```bash
# 安裝依賴
go mod download

# 運行服務
go run .

# 運行測試
go test ./...

# 生成 API 文檔
swag init
```

### Docker 構建
```bash
# 構建鏡像
docker build -t account-manager .

# 運行容器
docker run -p 8001:8001 \
  -e DATABASE_URL="postgres://..." \
  -e REDIS_URL="redis://..." \
  account-manager
```

## API 文檔

服務啟動後，可以通過以下地址訪問 Swagger API 文檔：
- http://localhost:8001/swagger/index.html

## 安全性

### 認證
- 使用 JWT 令牌進行 API 認證
- 支持令牌刷新機制
- 實現令牌黑名單防止重放攻擊

### 數據保護
- 密碼等敏感信息加密存儲
- 使用 HTTPS 進行數據傳輸
- 實現 CORS 保護

### 輸入驗證
- 所有輸入數據進行嚴格驗證
- 防止 SQL 注入攻擊
- 實現速率限制

## 監控

### 健康檢查
服務提供健康檢查端點，用於：
- 容器健康狀態監控
- 負載均衡器健康檢查
- 服務發現

### 日誌
- 結構化日誌輸出
- 不同級別的日誌記錄
- 敏感信息過濾

### 指標
- 請求計數和延遲
- 錯誤率統計
- 數據庫連接狀態
- Redis 連接狀態

## 故障排除

### 常見問題

1. **數據庫連接失敗**
   - 檢查 DATABASE_URL 配置
   - 確認數據庫服務運行正常
   - 驗證網絡連接

2. **Redis 連接失敗**
   - 檢查 REDIS_URL 配置
   - 確認 Redis 服務運行正常
   - 驗證認證信息

3. **Bluesky 認證失敗**
   - 檢查帳號憑據
   - 驗證代理配置
   - 確認網絡連接

### 日誌分析
```bash
# 查看服務日誌
docker logs account-manager

# 實時監控日誌
docker logs -f account-manager

# 過濾錯誤日誌
docker logs account-manager 2>&1 | grep ERROR
```

## 貢獻

1. Fork 項目
2. 創建功能分支
3. 提交更改
4. 推送到分支
5. 創建 Pull Request

## 許可證

MIT License
