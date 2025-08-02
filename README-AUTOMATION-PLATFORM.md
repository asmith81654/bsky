# Bluesky Automation Platform

一個基於 Docker 的大規模 Bluesky 帳號自動化管理平台，支持 50+ 帳號的智能化社交媒體運營。

## 🚀 功能特性

### 大規模帳號管理
- 支持管理 50+ 個 Bluesky 帳號
- 動態擴展，可隨時添加新帳號
- 每個帳號獨立身份和配置
- 帳號間完全隔離，互不影響

### 代理網絡隔離
- 每個帳號使用不同的代理服務器
- 確保網絡流量路由完全獨立
- 避免帳號間網絡關聯性被檢測
- 支持 HTTP/SOCKS5 代理

### 多樣化自動化策略
- **內容發布策略**: 定時發帖、話題跟隨、內容轉發
- **社交互動策略**: 自動關注、點讚、回覆、轉發
- **監控策略**: 關鍵詞監控、競爭對手分析、趨勢追蹤
- **增長策略**: 粉絲獲取、影響力擴展、社群建設

### 並發自動化操作
- 多帳號同時執行不同自動化任務
- 支援各種 Bluesky 操作
- 任務長時間運行，互不干擾
- 支持複雜工作流程和任務鏈

### 智能管理和監控
- 批量管理所有帳號
- 監控每個帳號運行狀態和策略效果
- 靈活任務調度和資源分配
- 策略執行數據分析和優化建議

## 🏗️ 系統架構

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Dashboard │    │   API Gateway   │    │   Monitoring    │
│     (React)     │    │    (Port 8000)  │    │   (Port 8005)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────┐
    │                            │                            │
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Account   │  │    Proxy    │  │  Strategy   │  │    Task     │
│  Manager    │  │   Manager   │  │   Engine    │  │  Scheduler  │
│ (Port 8001) │  │ (Port 8002) │  │ (Port 8003) │  │ (Port 8004) │
└─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
         │                │                │                │
         └────────────────┼────────────────┼────────────────┘
                          │                │
                ┌─────────────────┐  ┌─────────────────┐
                │   PostgreSQL    │  │      Redis      │
                │   (Database)    │  │     (Cache)     │
                └─────────────────┘  └─────────────────┘
                          │
                ┌─────────────────┐
                │     Workers     │
                │   (Scalable)    │
                └─────────────────┘
```

## 🛠️ 技術棧

- **後端**: Go + Gin/Echo 框架
- **數據庫**: PostgreSQL + Redis
- **容器化**: Docker + Docker Compose
- **前端**: React + TypeScript + Ant Design
- **代理**: HTTP/SOCKS5 代理支持
- **監控**: 自定義監控和統計系統

## 📦 快速開始

### 前置要求
- Docker 20.0+
- Docker Compose 2.0+
- 至少 4GB RAM
- 至少 10GB 可用磁盤空間

### 安裝步驟

1. **克隆項目**
```bash
git clone <repository-url>
cd bsky-automation-platform
```

2. **配置環境變量**
```bash
cp .env.example .env
# 編輯 .env 文件，設置密碼和配置
```

3. **啟動服務**
```bash
# 啟動所有服務
docker-compose up -d

# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs -f
```

4. **訪問管理界面**
- Web Dashboard: http://localhost:3000
- API Gateway: http://localhost:8000
- API 文檔: http://localhost:8000/swagger

### 基本使用

1. **添加帳號**
```bash
curl -X POST http://localhost:8000/api/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "handle": "your-handle.bsky.social",
    "password": "your-password",
    "proxy_id": 1
  }'
```

2. **配置代理**
```bash
curl -X POST http://localhost:8000/api/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Proxy 1",
    "type": "http",
    "host": "proxy.example.com",
    "port": 8080,
    "username": "user",
    "password": "pass"
  }'
```

3. **創建策略**
```bash
curl -X POST http://localhost:8000/api/strategies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自動關注策略",
    "type": "follow",
    "config": {
      "keywords": ["AI", "科技"],
      "daily_limit": 50,
      "delay_range": [300, 600]
    },
    "schedule": "0 */2 * * *"
  }'
```

## 📚 文檔

- [API 文檔](./docs/api.md)
- [策略配置指南](./docs/strategies.md)
- [部署指南](./docs/deployment.md)
- [故障排除](./docs/troubleshooting.md)

## 🔧 開發

### 開發環境設置
```bash
# 啟動開發環境
docker-compose -f docker-compose.dev.yml up -d

# 運行測試
make test

# 代碼格式化
make fmt

# 構建所有服務
make build
```

### 項目結構
```
bsky-automation-platform/
├── services/              # 微服務
│   ├── account-manager/   # 帳號管理服務
│   ├── proxy-manager/     # 代理管理服務
│   ├── strategy-engine/   # 策略引擎服務
│   ├── task-scheduler/    # 任務調度服務
│   ├── monitoring/        # 監控服務
│   ├── api-gateway/       # API 網關
│   └── worker/            # 工作容器
├── web-dashboard/         # Web 管理界面
├── shared/                # 共享庫
├── configs/               # 配置文件
├── scripts/               # 部署腳本
└── docs/                  # 文檔
```

## 🚨 注意事項

1. **合規使用**: 請確保遵守 Bluesky 的服務條款和使用政策
2. **代理配置**: 建議使用高質量的代理服務器以確保穩定性
3. **速率限制**: 系統內置智能速率限制，避免觸發平台限制
4. **數據安全**: 請妥善保管帳號密碼和 API 密鑰
5. **監控告警**: 建議設置監控告警以及時發現問題

## 📄 許可證

MIT License

## 🤝 貢獻

歡迎提交 Issue 和 Pull Request！

## 📞 支持

如有問題，請提交 Issue 或聯繫維護團隊。
