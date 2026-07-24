# HTTP API 设计

## 1. 通用约定

- API 前缀：`/api/v1`。
- JSON 字段使用 `camelCase`，时间使用 UTC RFC 3339。
- 所有响应使用统一信封：

```json
{
  "data": {},
  "error": null,
  "requestId": "01J..."
}
```

失败时 `data` 为 `null`，`error` 包含稳定 `code`、用户消息和可选安全详情。

## 2. 认证面

### 运营 Web

使用 `wildman_session` HttpOnly Cookie、Origin 白名单和双提交 CSRF Token。现有接口：

```text
GET  /api/v1/setup/status
POST /api/v1/setup/admin
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

### 野人音乐客户端

使用独立 Bearer Token：

```http
Authorization: Bearer wm_live_<prefix>_<secret>
```

`prefix` 固定为 11 个无填充 Base64URL 字符，`secret` 固定为 43 个无填充 Base64URL 字符（256 bit 随机秘密）。完整 Token 只在创建时返回一次。Token 区分大小写且不做空白或大小写归一化；客户端 API 不使用运营 Session 代替设备身份。

Bearer 认证只接受单个 `Authorization` 请求头和单个不含空白的 Token；认证方案名 `Bearer` 不区分大小写。格式错误或未知 Token 返回 `401 CLIENT_AUTH_REQUIRED`，已撤销 Token 返回 `401 CLIENT_REVOKED`，两者均携带 `WWW-Authenticate: Bearer`。认证得到的客户端内部 ID 写入服务端请求上下文，不接受请求体覆盖。

## 3. 系统接口

```text
GET /api/v1/health
GET /api/v1/ready
GET /api/v1/system/info
```

中央服务不提供本地音乐目录或 ffprobe 状态。

## 4. 客户端凭证 API

以下接口只允许运营管理员调用：

```text
POST /api/v1/clients
GET  /api/v1/clients
POST /api/v1/clients/{clientId}/revoke
POST /api/v1/clients/{clientId}/delete
```

创建请求：

```json
{
  "name": "Alice NAS"
}
```

名称去除首尾空白后必须包含 1 至 100 个字符。创建与撤销请求必须同时携带有效运营 Session 和 `X-CSRF-Token`；列表请求必须携带有效运营 Session。创建成功返回 `201 Created`：

```json
{
  "data": {
    "client": {
      "id": "client-id",
      "name": "Alice NAS",
      "tokenPrefix": "AbCdEfGhI_k",
      "status": "active",
      "lastSeenAt": null,
      "revokedAt": null,
      "createdAt": "2026-07-23T12:00:00Z"
    },
    "token": "wm_live_AbCdEfGhI_k_<secret>"
  },
  "error": null,
  "requestId": "http-request-id"
}
```

`data.token` 只在创建响应出现一次。列表返回 `{ "clients": [...] }`，不会返回 `token` 或 `tokenHash`。撤销成功返回当前客户端记录；重复撤销成功且保留首次 `revokedAt`，不存在的 `clientId` 返回 404。

删除请求正文为 `{ "name": "Alice NAS" }`，只允许删除已撤销且名称完全匹配的客户端；删除其观测、解析任务、候选和审核结果，保留不含敏感详情的审计事件。创建、撤销、删除均写入运营审计。

运营管理员可调用 `POST /api/v1/operations/retention` 执行保留清理，请求需要 Session 与 CSRF。

## 5. 曲目解析 API

### 创建解析请求

```http
POST /api/v1/resolutions
Authorization: Bearer wm_live_...
Idempotency-Key: 019f...
Content-Type: application/json
```

```json
{
  "clientTrackId": "local-track-123",
  "fileName": "02 - 七里香.flac",
  "title": "七里香",
  "artists": ["周杰伦"],
  "album": "",
  "discNumber": 1,
  "trackNumber": 2,
  "durationMs": 299520,
  "format": "flac",
  "fingerprint": null,
  "observedAt": "2026-07-23T12:00:00Z"
}
```

输入限制：

- `clientTrackId` 去除首尾空白后必须非空，最多 200 个字符。
- `fileName` 可为空，非空时最多 255 个字符，且只能是文件名，不能包含绝对路径、目录或路径遍历。
- `title` 与 `album` 分别最多 500 个字符。
- `discNumber` 与 `trackNumber` 可为空，非空时必须是 1 至 999 的整数。
- `artists` 最多 32 项；每项去除首尾空白后必须非空且最多 200 个字符。
- `durationMs` 可为空，非空时必须在 0 至 7 天之间。
- `format` 最多 32 个字符，`fingerprint` 最多 16 KiB，`observedAt` 必填。

接口同时拒绝未知 JSON 字段和音频正文。任一观测字段不符合限制时统一返回 `422 OBSERVATION_INVALID`，不会回显完整请求正文。

成功返回 `202 Accepted`：

```json
{
  "data": {
    "requestId": "res_01J...",
    "status": "queued"
  },
  "error": null,
  "requestId": "http-request-id"
}
```

### 查询结果

```text
GET /api/v1/resolutions/{requestId}
```

只有创建该请求的客户端可以读取；不存在和属于其他客户端的请求统一返回 `404 RESOLUTION_NOT_FOUND`。排队期间返回空 `candidates`，匹配完成后返回有序候选：

```json
{
  "data": {
    "requestId": "res_01J...",
    "status": "matched",
    "candidates": [
      {
        "recordingId": "rec_01J...",
        "score": 0.964,
        "title": "七里香",
        "artists": ["周杰伦"],
        "release": "七里香",
        "source": "musicbrainz",
        "sources": ["musicbrainz", "wikidata"],
        "evidence": ["标题完全一致", "歌手完全一致"],
        "conflicts": [],
        "tagPatch": [
          {
            "field": "album",
            "current": { "text": "" },
            "suggested": { "text": "七里香" },
            "source": "musicbrainz"
          }
        ]
      }
    ]
  },
  "error": null,
  "requestId": "http-request-id"
}
```

`tagPatch` 只提供字段建议；Wildman Service 不读取或写入 NAS 文件。野人音乐必须在本地展示差异、获得用户确认后再安全写回并复验。

### 审核与本地写回回报

```http
POST /api/v1/resolutions/{requestId}/review
Authorization: Bearer wm_live_...
Content-Type: application/json
```

```json
{
  "decision": "accepted",
  "recordingId": "rec_01J...",
  "writebackStatus": "succeeded",
  "writebackErrorCode": ""
}
```

`decision` 为 `accepted` 或 `rejected`。接受时 `recordingId` 必须属于该请求候选；写回状态为 `not_attempted`、`succeeded` 或 `failed`。失败只发送最多 64 个可打印 ASCII 字符的稳定错误码，不发送路径、异常堆栈或文件内容。拒绝时写回状态必须为 `not_attempted`。

## 6. 幂等、限流与错误

- 创建解析请求必须带 `Idempotency-Key`；值为 1 至 200 个可打印 ASCII 字符，不能包含空白。同一客户端重复键返回原请求，不同客户端可复用相同键。
- 限流按认证后的客户端而不是代理 IP 计算。单节点 MVP 使用每客户端每分钟 120 次的进程内固定窗口；超限返回 `429 CLIENT_RATE_LIMITED` 和整数秒 `Retry-After`，服务重启后窗口重置。
- 成功认证时更新 `lastSeenAt`，为避免高频写库，同一客户端最多每 5 分钟持久化一次。
- Provider 的 429 不直接透传客户端，映射为稳定状态并遵循服务端退避。

基础错误码：

| code | HTTP | 含义 |
|---|---:|---|
| `CLIENT_AUTH_REQUIRED` | 401 | 缺少或无效客户端 Token |
| `CLIENT_REVOKED` | 401 | 客户端已撤销 |
| `CLIENT_NAME_INVALID` | 422 | 客户端安装名称不符合限制 |
| `CLIENT_NOT_FOUND` | 404 | 运营端指定的客户端不存在 |
| `CLIENT_DELETE_NOT_ALLOWED` | 409 | 客户端未撤销或确认名称不匹配 |
| `OBSERVATION_INVALID` | 422 | 曲目观测不符合限制 |
| `IDEMPOTENCY_KEY_REQUIRED` | 400 | 缺少幂等键 |
| `IDEMPOTENCY_KEY_INVALID` | 400 | 幂等键格式无效 |
| `CONTENT_TYPE_UNSUPPORTED` | 415 | 解析请求正文不是 JSON |
| `REVIEW_INVALID` | 422 | 审核或本地写回结果无效 |
| `RESOLUTION_NOT_FOUND` | 404 | 请求不存在或不属于当前客户端 |
| `CLIENT_RATE_LIMITED` | 429 | 客户端超过配额 |
| `PROVIDER_UNAVAILABLE` | 503 | 上游暂时不可用 |
| `ACCOUNT_AUTH_REQUIRED` | 401 | 需要有效账户 Session |
| `ACCOUNT_QUOTA_EXCEEDED` | 429 | 账户本月解析额度用尽 |
| `DEVICE_AUTH_PENDING` | 428 | 设备授权等待用户批准 |
| `DEVICE_AUTH_CONSUMED` | 410 | 设备 Token 已领取 |
| `SUBSCRIPTION_INVALID` | 422 | 账户或订阅参数无效 |

## 7. 演进

- 野人音乐发送客户端版本和 API 版本，服务端在一个主版本内保持向后兼容。
- 新增字段默认可忽略；删除或改变语义进入新的 API 主版本。
- OpenAPI 在首个真实野人音乐接入切片稳定后纳入构建。

## 8. 账户、设备授权与配额

```text
POST /api/v1/accounts/register
POST /api/v1/accounts/login
POST /api/v1/device/authorizations
POST /api/v1/account/device/approve
POST /api/v1/device/token
GET  /api/v1/accounts                           # 运营管理员
POST /api/v1/accounts/{accountId}/subscription  # 运营管理员
```

注册与登录请求为 `{ "email": "user@example.com", "password": "..." }`，成功返回一次账户 Session Token。账户 Session 仅用于批准设备，不是运营 Session 或客户端凭证。

设备开始请求 `{ "clientName": "Alice NAS" }`；批准请求使用 `Authorization: Bearer wa_session_...` 并提交 `{ "userCode": "..." }`；设备轮询提交 `{ "deviceCode": "..." }`。批准后完整 `wm_live_` Token 只返回一次。

订阅变更请求需要运营 Session 和 CSRF：

```json
{ "plan": "pro", "status": "active", "monthlyQuota": 10000 }
```

账户客户端创建解析任务时按 UTC 月消费额度；相同 `Idempotency-Key` 重试只计一次。超额返回 `429 ACCOUNT_QUOTA_EXCEEDED`。支付卡和自动扣款不在当前服务边界，详见 [ACCOUNTS_BILLING.md](./ACCOUNTS_BILLING.md)。
