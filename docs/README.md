# Wildman 项目文档

本目录是 Wildman Service 的产品与工程基线。产品范围、架构边界和开发优先级发生变化时，应先更新对应文档，再开始实现。

## 文档索引

| 文档 | 用途 |
|---|---|
| [PRODUCT.md](./PRODUCT.md) | 产品定位、目标用户、核心流程、范围与成功指标 |
| [MVP.md](./MVP.md) | 首个可发布版本的功能边界、验收标准和非目标 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 中央控制面、野人音乐数据面、缓存与匹配边界 |
| [DATA_MODEL.md](./DATA_MODEL.md) | 核心领域对象、关系、状态和数据来源追踪方案 |
| [API.md](./API.md) | HTTP API 设计规范、错误格式和 MVP 接口清单 |
| [CLIENT_INTEGRATION.md](./CLIENT_INTEGRATION.md) | 野人音乐凭证、观测提交、重试与版本兼容接入说明 |
| [PROVIDER_MUSICBRAINZ.md](./PROVIDER_MUSICBRAINZ.md) | 首个 Provider 的认证、限流、缓存、许可与再分发边界 |
| [PROVIDER_WIKIDATA.md](./PROVIDER_WIKIDATA.md) | 第二 Provider 与跨来源证据边界 |
| [PROVIDER_ACOUSTID.md](./PROVIDER_ACOUSTID.md) | 本地 Chromaprint 与中央 AcoustID 查询边界 |
| [MATCH_QUALITY.md](./MATCH_QUALITY.md) | 脱敏匹配样本、离线质量指标与扩充规则 |
| [BACKUP_RESTORE.md](./BACKUP_RESTORE.md) | PostgreSQL 备份、验证、恢复与演练步骤 |
| [RELEASE.md](./RELEASE.md) | 反向代理、HTTPS、双架构镜像与发布检查 |
| [SECURITY_CHECKS.md](./SECURITY_CHECKS.md) | 漏洞、秘密、许可检查与 SPDX SBOM 流程 |
| [MUSICBRAINZ_DUMP.md](./MUSICBRAINZ_DUMP.md) | 开放目录 Dump 的许可、转换、导入与规模边界 |
| [POSTGRES_WORKER.md](./POSTGRES_WORKER.md) | PostgreSQL 配置、迁移边界与独立 Worker 领取流程 |
| [ACCOUNTS_BILLING.md](./ACCOUNTS_BILLING.md) | 公开账户、设备授权、月配额与计费控制面边界 |
| [SECURITY.md](./SECURITY.md) | 鉴权、文件安全、第三方接口、隐私和合规要求 |
| [ROADMAP.md](./ROADMAP.md) | 从当前骨架到稳定版本的阶段性路线 |
| [TODO.md](./TODO.md) | 可直接执行和勾选的开发任务列表 |
| [REFERENCES.md](./REFERENCES.md) | beets、Picard、Lidarr 的参考点与明确取舍 |

## 约定

- `P0`：MVP 发布前必须完成。
- `P1`：MVP 后优先完成，影响主要使用体验。
- `P2`：增强功能，不阻塞核心闭环。
- Wildman Service 不执行文件写入；字段建议由野人音乐本地审核和应用。
- 第三方平台返回的是来源观测数据，不直接等同于最终可信数据。
- 项目不提供音乐文件下载、破解会员内容或绕过平台权限的能力。

## 当前阶段

项目已从 NAS 本地扫描服务调整为中央元数据服务。当前执行 Phase 1：运营者签发客户端凭证，野人音乐提交最小曲目观测，并创建可查询、幂等且按客户端隔离的解析请求。
