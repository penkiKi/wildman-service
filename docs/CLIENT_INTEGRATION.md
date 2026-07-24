# 野人音乐客户端接入

本文说明野人音乐如何接入当前 Wildman Service API v1。完整字段与错误信封以 [API.md](./API.md) 为准。

## 1. 获取与保存凭证

运营管理员在 Wildman Service 的“客户端”页面为每个野人音乐安装创建独立凭证。完整 Token 只显示一次，格式为：

```text
wm_live_<prefix>_<secret>
```

野人音乐必须将 Token 写入操作系统或应用提供的秘密存储，不写入日志、错误报告、数据库导出或普通配置界面。Token 泄露或设备停用后，由运营管理员撤销旧客户端并签发新凭证；已撤销 Token 不能恢复。

生产接入只使用 HTTPS。中央服务不需要也不应获得 NAS 挂载、音乐目录、野人音乐数据库或文件读取权限。

## 2. 请求约定

API 主版本位于路径 `/api/v1`。每次客户端请求携带：

```http
Authorization: Bearer wm_live_...
Accept: application/json
User-Agent: WildmanMusic/1.2.3
```

创建解析请求还必须携带：

```http
Content-Type: application/json
Idempotency-Key: <本次逻辑请求的稳定唯一键>
```

`Idempotency-Key` 在同一安装内保持唯一，重试同一逻辑请求时必须复用原值；新建解析任务时生成新值。值为 1 至 200 个不含空白的可打印 ASCII 字符。服务端限流按认证后的安装计算，当前为每分钟 120 次。

## 3. 提交最小观测

野人音乐在本地完成扫描、标签读取和技术参数提取，只提交匹配需要的最小事实：

```http
POST /api/v1/resolutions HTTP/1.1
Authorization: Bearer wm_live_...
Content-Type: application/json
Idempotency-Key: 019f8f25-1a70-7d2d-a780-7d5e57658592

{
  "clientTrackId": "local-track-123",
  "fileName": "02 - 七里香.flac",
  "title": "七里香",
  "artists": ["周杰伦"],
  "album": "七里香",
  "durationMs": 299520,
  "format": "flac",
  "fingerprint": null,
  "observedAt": "2026-07-23T12:00:00Z"
}
```

`clientTrackId` 由野人音乐生成并在该安装内稳定。`fileName` 只能是基名；不得发送绝对路径、相对目录、文件正文、完整标签转储、任意 URL 或数据库记录。服务端拒绝未知 JSON 字段，具体长度限制见 [API.md](./API.md#5-曲目解析-api)。

成功响应为 `202 Accepted`：

```json
{
  "data": {
    "requestId": "opaque-request-id",
    "status": "queued"
  },
  "error": null,
  "requestId": "http-request-id"
}
```

网络超时或连接中断后，使用相同 `Idempotency-Key` 重试；服务端返回首次创建的解析请求，不会创建第二个任务。

## 4. 查询状态

使用创建响应中的 `requestId` 查询：

```http
GET /api/v1/resolutions/<requestId>
Authorization: Bearer wm_live_...
```

当前可能状态为 `queued`、`matching`、`matched`、`no_match` 或 `failed`。轮询应采用退避并遵守 `429` 的 `Retry-After`，不高频固定间隔请求。只有创建任务的安装可查询；不存在和不属于当前安装的 ID 都返回 `404 RESOLUTION_NOT_FOUND`。

## 5. 错误与重试

- `400`、`415`、`422`：修正请求后再提交，不原样自动重试。
- `401 CLIENT_AUTH_REQUIRED`：检查本地凭证配置；不要把 Token 写入诊断日志。
- `401 CLIENT_REVOKED`：停止请求，提示重新签发凭证。
- `404 RESOLUTION_NOT_FOUND`：停止轮询该请求。
- `429 CLIENT_RATE_LIMITED`：等待 `Retry-After` 指定的秒数。
- `5xx` 或网络错误：使用指数退避和抖动重试；创建请求继续复用原幂等键。

诊断信息只记录 HTTP Request ID、解析请求 ID、状态码和稳定错误码，不记录 Authorization、完整观测正文或 NAS 路径。

## 6. 版本兼容

- API 主版本由 URL 表达；当前为 `/api/v1`。客户端版本通过标准 `User-Agent: WildmanMusic/<semver>` 发送。
- v1 内可以新增可选响应字段、候选证据或新错误码。客户端必须忽略不认识的响应字段，并把未知错误码按对应 HTTP 状态的通用策略处理。
- v1 不会删除既有字段、改变字段语义或把可选字段改为必填。此类破坏性变化进入新的 `/api/v2`，并提供明确迁移期。
- 请求端继续拒绝未知字段，因此野人音乐只能发送其目标 API 主版本已定义的字段。
- 客户端升级不能复用新的逻辑请求键；只有同一请求的传输重试才复用 `Idempotency-Key`。

运营管理员手动签发仍是 MVP 默认接入方式；P1 同时提供独立客户账户与设备授权流程，详见 [ACCOUNTS_BILLING.md](./ACCOUNTS_BILLING.md)。自动 Token 轮换仍未提供。
