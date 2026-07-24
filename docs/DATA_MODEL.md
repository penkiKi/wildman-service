# 数据模型

## 1. 模型原则

- 客户端观测、Provider 来源观测、中央规范目录和解析结果分开保存。
- 不保存 NAS 绝对路径、完整音乐文件、播放历史或野人音乐数据库。
- Provider payload 不直接覆盖规范实体。
- Token 只保存摘要；客户端身份只能来自认证上下文。
- 时间统一使用 UTC RFC 3339，ID 为不透明字符串。

## 2. 运营身份

### User / Session

现有 `users` 和 `sessions` 只用于运营 Web 管理员。MVP 不提供最终用户公开注册。

### ClientInstallation

代表一个获得授权的野人音乐安装实例。

| 字段 | 说明 |
|---|---|
| `id` | 客户端内部 ID |
| `name` | 运营者可识别名称 |
| `token_prefix` | 非敏感 Token 前缀 |
| `token_hash` | 完整 Token 的 SHA-256 摘要 |
| `status` | `active` 或 `revoked` |
| `created_by_user_id` | 签发凭证的运营管理员 |
| `last_seen_at` / `revoked_at` | 使用与撤销时间 |

新安装只能以 `active` 状态创建，且 `revoked_at` 为空。撤销是单向且幂等的状态变更：首次撤销同时写入 `status = revoked` 与 UTC `revoked_at`，重复撤销保留首次时间，不允许恢复为 `active`。只有状态为 `active` 且 `revoked_at` 为空的安装可以通过认证。

`last_seen_at` 记录最近一次成功 Bearer 认证的近似时间；为限制写放大，同一客户端最多每 5 分钟持久化一次，因此不作为精确审计日志。

### Customer Account / Subscription / DeviceAuthorization

客户账户与运营 `users` 隔离。订阅保存方案、状态和月额度；月用量按 UTC `YYYY-MM` 聚合，`quota_consumptions` 以客户端与幂等键去重。设备授权码有效 10 分钟且单次消费，数据库只保存设备码和用户码摘要。账户批准后签发的 `ClientInstallation.account_id` 建立配额归属。

完整 Token 格式固定为 `wm_live_<prefix>_<secret>`：`prefix` 是 8 个随机字节的无填充 Base64URL（11 字符），`secret` 是 32 个随机字节的无填充 Base64URL（43 字符）。`token_prefix` 只保存 11 字符的 `prefix` 段；`token_hash` 保存完整 Token UTF-8 字节的 SHA-256 摘要，并使用无填充 Base64URL 编码（43 字符）。完整 Token 只在创建时返回一次，不持久化、不记录日志，格式不正确的 Token 在查询数据库前拒绝。

## 3. 客户端观测

### TrackObservation

野人音乐对本地曲目的最小事实快照。

| 字段 | 说明 |
|---|---|
| `client_id` | 来自 Bearer Token 的客户端身份 |
| `client_track_id` | 客户端生成的稳定曲目 ID |
| `file_name` | 可选文件名，不含绝对目录 |
| `title` / `artists_json` / `album` | 当前标签 |
| `disc_number` / `track_number` | 可选发行内碟号与曲号 |
| `duration_ms` / `format` | 本地技术事实 |
| `fingerprint` | 可选音频指纹，不是音频正文 |
| `payload_hash` | 观测内容摘要，用于变化判断 |
| `observed_at` | 客户端观测时间 |

唯一约束：`client_id + client_track_id`。

## 4. 中央规范目录

### Artist

保存名称、规范化名称和可选排序名称。规范化名称用于检索，不替代用户可见原文。

### Release

代表专辑、单曲或其他发行，包含标题、发行日期和可选条码。艺术家使用有序关联表表达。

### Recording

代表歌曲录音身份，包含标题、时长和可选 ISRC。它与具体发行中的曲目位置分离。

### Track

连接 Release 与 Recording，保存碟号、曲号、发行内标题和时长。

## 5. 来源观测

### SourceObservation

| 字段 | 说明 |
|---|---|
| `provider` / `external_id` | 来源和外部 ID |
| `entity_type` | artist、release 或 recording |
| `canonical_entity_id` | 可选规范实体关联 |
| `payload_json` / `payload_hash` | 原始响应和不可变摘要 |
| `fetched_at` / `expires_at` | 抓取与刷新边界 |
| `adapter_version` | 解析该 payload 的适配器版本 |

同一 Provider 的新响应创建新观测，不覆盖历史 payload。

## 6. 解析请求

### ResolutionRequest

| 字段 | 说明 |
|---|---|
| `client_id` / `observation_id` | 客户端和输入观测 |
| `idempotency_key` | 客户端重试去重键 |
| `status` | queued、matching、matched、no_match 或 failed |
| `last_error_code` | 稳定错误码，不保存敏感上游正文 |

唯一约束：`client_id + idempotency_key`。

### ResolutionCandidate

关联规范 Recording，保存排名、0–1 分数、字段证据和冲突。候选是一次请求的结果，不直接改变 TrackObservation 或中央目录。

## 7. 迁移说明

### 数据保留与审计

- 已结束解析请求、候选与审核结果保留 180 天；孤立观测按相同期限清理。
- 已过期且不再被候选引用的来源观测在过期 30 天后清理。
- 审计事件只记录操作者内部 ID、动作、主体类型/ID和时间，不记录 Token、名称、路径或 payload，保留 365 天。
- 客户端只有撤销并确认名称后才能删除；关联客户端数据在单个事务中删除，删除审计保留。

`001_initial.sql` 中的 libraries、media_files、tag_snapshots 和 library_issues 来自旧的本地扫描方向。迁移保持不可变，但中央服务不再使用这些表；正式数据清理在发布前通过新迁移完成，不修改历史迁移。
