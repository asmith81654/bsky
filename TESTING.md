# Testing Guide

本指南說明如何測試 Bluesky Automation Platform 的服務。

## 快速開始

### 1. 啟動測試環境

```bash
# 啟動所有服務
./scripts/start-test-env.sh

# 啟動服務並查看日誌
./scripts/start-test-env.sh --logs
```

### 2. 運行快速測試

```bash
# 基本健康檢查和 API 測試
./scripts/quick-test.sh
```

### 3. 運行完整測試

```bash
# 完整的 API 功能測試
./scripts/test-services.sh
```

### 4. 停止測試環境

```bash
# 停止服務（保留數據）
./scripts/stop-test-env.sh

# 停止服務並清理數據
./scripts/stop-test-env.sh --clean

# 完全清理（包括鏡像）
./scripts/stop-test-env.sh --full-clean
```

## 服務端點

### Account Manager (端口 8001)
- **API Base URL**: http://localhost:8001/api/v1
- **Health Check**: http://localhost:8001/health
- **API 文檔**: http://localhost:8001/swagger/index.html

#### 主要端點：
- `POST /auth/login` - 用戶登錄
- `GET /accounts` - 獲取帳號列表
- `POST /accounts` - 創建帳號
- `GET /accounts/{id}` - 獲取特定帳號
- `PUT /accounts/{id}` - 更新帳號
- `DELETE /accounts/{id}` - 刪除帳號

### Proxy Manager (端口 8002)
- **API Base URL**: http://localhost:8002/api/v1
- **Health Check**: http://localhost:8002/health
- **API 文檔**: http://localhost:8002/swagger/index.html

#### 主要端點：
- `GET /proxies` - 獲取代理列表
- `POST /proxies` - 創建代理
- `GET /proxies/{id}` - 獲取特定代理
- `PUT /proxies/{id}` - 更新代理
- `DELETE /proxies/{id}` - 刪除代理
- `POST /assignment/assign` - 分配代理

## 手動測試

### 1. 測試 Account Manager

#### 登錄獲取令牌：
```bash
curl -X POST http://localhost:8001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

#### 創建帳號：
```bash
# 使用上面獲取的 access_token
curl -X POST http://localhost:8001/api/v1/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "handle": "testuser.bsky.social",
    "password": "test_password",
    "host": "https://bsky.social"
  }'
```

#### 獲取帳號列表：
```bash
curl -X GET http://localhost:8001/api/v1/accounts \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### 2. 測試 Proxy Manager

#### 創建代理：
```bash
curl -X POST http://localhost:8002/api/v1/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Proxy",
    "type": "http",
    "host": "proxy.example.com",
    "port": 8080,
    "username": "user",
    "password": "pass"
  }'
```

#### 獲取代理列表：
```bash
curl -X GET http://localhost:8002/api/v1/proxies
```

#### 測試代理連接：
```bash
curl -X POST http://localhost:8002/api/v1/proxies/1/test
```

#### 分配代理給帳號：
```bash
curl -X POST http://localhost:8002/api/v1/assignment/assign \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "proxy_id": 1,
    "strategy": "manual"
  }'
```

## 數據庫訪問

### PostgreSQL
```bash
# 連接到數據庫
docker exec -it bsky-postgres-test psql -U bsky_user -d bsky_automation

# 查看表
\dt

# 查看帳號
SELECT id, handle, status FROM accounts;

# 查看代理
SELECT id, name, host, port, status FROM proxies;
```

### Redis
```bash
# 連接到 Redis
docker exec -it bsky-redis-test redis-cli -a redis_test_password

# 查看所有鍵
KEYS *

# 查看健康檢查數據
KEYS proxy_health:*

# 查看分配狀態
KEYS proxy_round_robin*
```

## 測試數據

測試環境會自動加載種子數據，包括：
- 5 個示例代理服務器
- 5 個示例帳號
- 5 個示例策略
- 示例任務和指標數據

## 故障排除

### 常見問題

1. **服務無法啟動**
   ```bash
   # 檢查 Docker 狀態
   docker ps
   
   # 查看服務日誌
   docker-compose -f docker-compose.test.yml logs
   ```

2. **數據庫連接失敗**
   ```bash
   # 檢查 PostgreSQL 狀態
   docker-compose -f docker-compose.test.yml logs postgres
   
   # 測試數據庫連接
   docker exec bsky-postgres-test pg_isready -U bsky_user
   ```

3. **Redis 連接失敗**
   ```bash
   # 檢查 Redis 狀態
   docker-compose -f docker-compose.test.yml logs redis
   
   # 測試 Redis 連接
   docker exec bsky-redis-test redis-cli -a redis_test_password ping
   ```

4. **API 返回 500 錯誤**
   ```bash
   # 查看應用日誌
   docker-compose -f docker-compose.test.yml logs account-manager
   docker-compose -f docker-compose.test.yml logs proxy-manager
   ```

### 重置環境

如果遇到問題，可以完全重置測試環境：

```bash
# 停止並清理所有內容
./scripts/stop-test-env.sh --full-clean

# 重新啟動
./scripts/start-test-env.sh
```

## 性能測試

### 基本負載測試

使用 `ab` (Apache Bench) 進行簡單的負載測試：

```bash
# 測試健康檢查端點
ab -n 100 -c 10 http://localhost:8001/health

# 測試帳號列表端點（需要先獲取令牌）
ab -n 50 -c 5 -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8001/api/v1/accounts
```

### 並發測試

測試代理健康檢查的並發性能：

```bash
# 創建多個代理後觀察健康檢查日誌
docker-compose -f docker-compose.test.yml logs -f proxy-manager | grep "health check"
```

## 集成測試

測試腳本 `test-services.sh` 執行以下集成測試：

1. **服務健康檢查** - 驗證所有服務正常運行
2. **認證流程** - 測試登錄和令牌管理
3. **帳號管理** - 測試完整的帳號 CRUD 操作
4. **代理管理** - 測試代理 CRUD 和分配功能
5. **統計端點** - 驗證統計數據正確性
6. **數據清理** - 確保測試數據正確清理

## 持續集成

在 CI/CD 環境中運行測試：

```bash
# CI 腳本示例
#!/bin/bash
set -e

# 啟動服務
./scripts/start-test-env.sh

# 等待服務就緒
sleep 30

# 運行測試
./scripts/test-services.sh

# 清理
./scripts/stop-test-env.sh --clean
```

## 監控和日誌

### 實時監控

```bash
# 監控所有服務日誌
docker-compose -f docker-compose.test.yml logs -f

# 監控特定服務
docker-compose -f docker-compose.test.yml logs -f account-manager

# 監控健康檢查
docker-compose -f docker-compose.test.yml logs -f proxy-manager | grep health
```

### 性能指標

```bash
# 查看容器資源使用
docker stats

# 查看服務響應時間
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8001/health
```

其中 `curl-format.txt` 內容：
```
     time_namelookup:  %{time_namelookup}\n
        time_connect:  %{time_connect}\n
     time_appconnect:  %{time_appconnect}\n
    time_pretransfer:  %{time_pretransfer}\n
       time_redirect:  %{time_redirect}\n
  time_starttransfer:  %{time_starttransfer}\n
                     ----------\n
          time_total:  %{time_total}\n
```
