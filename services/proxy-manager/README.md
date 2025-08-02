# Proxy Manager Service

代理管理服務負責管理代理服務器的 CRUD 操作、健康檢查、分配算法和性能監控。

## 功能特性

### 代理管理
- 創建、讀取、更新、刪除代理服務器配置
- 支持 HTTP 和 SOCKS5 代理類型
- 代理狀態管理（活躍、非活躍、錯誤）
- 代理連接測試和驗證

### 健康檢查
- 自動定期健康檢查
- 連接測試和響應時間監控
- 故障檢測和自動恢復
- 連續失敗處理

### 代理分配
- 智能代理分配算法
- 多種分配策略（自動、手動、輪詢、最少使用、最快響應）
- 負載均衡
- 代理池管理

### 統計和監控
- 代理使用統計
- 性能指標監控
- 健康狀態報告
- 響應時間分析

## API 端點

### 代理管理
- `GET /api/v1/proxies` - 獲取代理列表
- `POST /api/v1/proxies` - 創建新代理
- `GET /api/v1/proxies/{id}` - 獲取特定代理
- `PUT /api/v1/proxies/{id}` - 更新代理
- `DELETE /api/v1/proxies/{id}` - 刪除代理
- `POST /api/v1/proxies/{id}/test` - 測試代理連接
- `POST /api/v1/proxies/{id}/health-check` - 運行健康檢查

### 代理分配
- `GET /api/v1/assignment/available` - 獲取可用代理
- `POST /api/v1/assignment/assign` - 分配代理給帳號
- `POST /api/v1/assignment/release` - 釋放代理
- `GET /api/v1/assignment/usage` - 獲取代理使用情況

### 統計
- `GET /api/v1/stats/proxies` - 獲取代理統計
- `GET /api/v1/stats/health` - 獲取健康統計
- `GET /api/v1/stats/performance` - 獲取性能統計

### 健康檢查
- `GET /health` - 服務健康檢查

## 配置

### 環境變量
- `SERVICE_PORT` - 服務端口（默認：8002）
- `DATABASE_URL` - PostgreSQL 連接字符串
- `REDIS_URL` - Redis 連接字符串
- `ENVIRONMENT` - 運行環境（development/production）
- `PROXY_HEALTH_CHECK_INTERVAL` - 健康檢查間隔（秒，默認：300）
- `MAX_CONCURRENT_HEALTH_CHECKS` - 最大並發健康檢查數（默認：10）
- `MAX_PROXY_FAILURES` - 最大連續失敗次數（默認：3）

### 數據庫
服務需要連接到 PostgreSQL 數據庫，包含以下表：
- `proxies` - 代理服務器配置
- `accounts` - 帳號信息（用於分配關聯）

### Redis
用於：
- 健康檢查結果緩存
- 代理分配狀態
- 輪詢算法狀態
- 故障計數器
- 性能指標

## 代理分配策略

### 自動分配 (auto)
綜合考慮使用率和響應時間，選擇最佳代理。

### 手動分配 (manual)
指定特定的代理 ID 進行分配。

### 輪詢分配 (round_robin)
按順序輪流分配代理，確保負載均勻分布。

### 最少使用 (least_used)
選擇當前分配帳號數量最少的代理。

### 最快響應 (fastest)
選擇響應時間最短的代理。

## 健康檢查機制

### 檢查流程
1. 定期掃描所有活躍代理
2. 並發執行連接測試
3. 記錄響應時間和成功率
4. 更新代理健康狀態
5. 處理連續失敗

### 故障處理
- 連續失敗達到閾值時標記為錯誤狀態
- 自動從分配池中移除故障代理
- 恢復後自動重新加入分配池
- 發送故障告警通知

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
docker build -t proxy-manager .

# 運行容器
docker run -p 8002:8002 \
  -e DATABASE_URL="postgres://..." \
  -e REDIS_URL="redis://..." \
  proxy-manager
```

## API 文檔

服務啟動後，可以通過以下地址訪問 Swagger API 文檔：
- http://localhost:8002/swagger/index.html

## 使用示例

### 創建代理
```bash
curl -X POST http://localhost:8002/api/v1/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Proxy Server 1",
    "type": "http",
    "host": "proxy.example.com",
    "port": 8080,
    "username": "user",
    "password": "pass"
  }'
```

### 測試代理
```bash
curl -X POST http://localhost:8002/api/v1/proxies/1/test
```

### 分配代理
```bash
curl -X POST http://localhost:8002/api/v1/assignment/assign \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "strategy": "auto"
  }'
```

### 獲取統計
```bash
curl http://localhost:8002/api/v1/stats/proxies
```

## 監控

### 健康檢查
服務提供健康檢查端點，用於：
- 容器健康狀態監控
- 負載均衡器健康檢查
- 服務發現

### 日誌
- 結構化日誌輸出
- 健康檢查結果記錄
- 代理分配操作日誌
- 錯誤和告警日誌

### 指標
- 代理總數和狀態分布
- 健康檢查成功率
- 平均響應時間
- 分配操作統計

## 故障排除

### 常見問題

1. **代理連接失敗**
   - 檢查代理服務器配置
   - 驗證網絡連接
   - 確認認證信息

2. **健康檢查失敗**
   - 檢查代理服務器狀態
   - 驗證健康檢查 URL
   - 確認超時設置

3. **分配失敗**
   - 檢查可用代理數量
   - 驗證帳號存在性
   - 確認代理狀態

### 日誌分析
```bash
# 查看服務日誌
docker logs proxy-manager

# 實時監控日誌
docker logs -f proxy-manager

# 過濾健康檢查日誌
docker logs proxy-manager 2>&1 | grep "health check"
```

## 貢獻

1. Fork 項目
2. 創建功能分支
3. 提交更改
4. 推送到分支
5. 創建 Pull Request

## 許可證

MIT License
