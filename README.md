# Wildman Service

Wildman Service 是为野人音乐提供共享目录、缓存和匹配能力的中央音乐元数据服务。野人音乐部署在用户 NAS，负责扫描本地文件、读取标签和安全写回；Wildman Service 由运营方部署，只接收最小曲目观测，返回带来源、分数和冲突说明的候选。

中央服务不访问用户 NAS，不读取野人音乐数据库，也不托管或分发音乐文件。

## 技术栈

- Go 1.26 + chi
- SQLite（单节点 MVP）
- React 19 + TypeScript + Vite
- Tailwind CSS
- Bun 1.3.5

## 本地开发

前端：

```powershell
cd frontend
bun install
bun run dev
```

后端：

```powershell
go run ./cmd/server
```

前端地址为 `http://localhost:5173`，Vite 将 `/api` 代理到 `http://127.0.0.1:8080`。

## 生产构建

```powershell
cd frontend
bun install
bun run build
cd ..
go build -trimpath -o bin/wildman-service.exe ./cmd/server
```

## Docker

```powershell
docker compose up --build
```

启动后访问 `http://localhost:8080`。容器只持久化 `/data`，不需要挂载用户音乐目录。

## 当前状态

已完成：

- Go/React/Bun 单体骨架和嵌入式 Web
- SQLite 迁移、统一 API 信封、健康与就绪接口
- 运营管理员初始化、登录、Session、CSRF 和 Origin 白名单
- 受保护的运营 Web 布局
- 客户端安装、中央目录、来源观测和解析请求数据库基础
- TrackObservation、Artist、Release、Recording 和 Candidate 领域模型
- 运营管理员创建、列表和撤销客户端安装凭证 API
- 运营端客户端管理页面与一次性完整 Token 展示
- 客户端 Bearer Token 认证、最后使用时间和每客户端限流
- 曲目观测 upsert 与幂等解析请求创建、隔离查询 API
- 客户账户、设备授权、月度配额与订阅控制面
- 服务端移除 `/music`、ffprobe 和本地文件写入依赖

当前阶段实现运营者签发客户端凭证，以及野人音乐提交曲目观测和创建幂等解析请求。详见 [docs/TODO.md](./docs/TODO.md)。

野人音乐接入当前 API 的凭证、重试和兼容约定见 [docs/CLIENT_INTEGRATION.md](./docs/CLIENT_INTEGRATION.md)。
