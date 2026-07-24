# 首个 Provider：MusicBrainz

## 1. 选择

MVP 首个外部元数据 Provider 选择 MusicBrainz Web Service v2，唯一允许的 API 主机为 `musicbrainz.org`，基础路径为 `/ws/2/`，只使用 HTTPS。

选择理由：MusicBrainz 提供 artist、release、recording 等与当前中央目录一致的公开实体，有明确的 Web Service 使用规则和数据许可，且不需要模拟浏览器、破解登录或调用非官方私有接口。

本决定不包含 Cover Art Archive、歌词、音频文件、用户标签全文或其他第三方链接目标；这些来源必须单独完成许可和缓存评估后才能接入。

## 2. 认证与身份

当前只使用无需账号的公开读取接口，不配置用户密码或 OAuth。每次请求必须发送可识别的 `User-Agent`，格式为：

```text
WildmanService/<version> (<operator contact URL or email>)
```

生产部署必须配置有效的运营方联系信息。不得伪造浏览器 User-Agent、轮换身份规避限流或使用用户的 MusicBrainz 凭证。

## 3. 请求与限流

- 全服务对 `musicbrainz.org` 的请求速率上限为每秒 1 次，而不是每客户端各 1 次。
- 仅调用 `/ws/2/artist`、`/ws/2/release` 和 `/ws/2/recording` 的 JSON 查询/读取端点。
- 单次查询使用必要的最小 `inc` 参数；不无差别遍历或批量抓取目录。
- 收到 429 时尊重 `Retry-After`，暂停 Provider 请求；不把上游限流直接透传成客户端配额。
- 401、403、持续 429、响应结构异常或许可/服务条款变化会触发停用检查，不通过增加并发、代理或备用域名绕过。

网络超时、响应大小、重定向和有限重试的实现边界由 C2104 落地。

## 4. 缓存策略

缓存键由规范化查询、实体类型、响应所用 include 参数和适配器版本组成。

| 内容 | 初始 TTL | 说明 |
|---|---:|---|
| 精确实体读取 | 30 天 | 外部 ID 已知，变化频率低 |
| 搜索结果 | 7 天 | 排名与目录内容可能变化 |
| 确认无结果 | 6 小时 | 避免重复空查询但允许较快发现新增数据 |
| 429 / 暂时性错误 | 不作数据缓存 | 只执行短期退避，不记录为无结果 |

原始 JSON 作为不可变 `SourceObservation` 保存时必须记录 payload hash、抓取时间、过期时间和适配器版本。新响应新增观测，不覆盖历史。不得把 MusicBrainz payload 直接视为用户已接受的最终标签。

## 5. 许可与再分发

MusicBrainz 核心数据包含 CC0 公共领域数据，也可能包含按 CC BY-SA 提供的补充数据。实现与运营必须遵循 MusicBrainz 当前公布的 [Database License](https://musicbrainz.org/doc/About/Data_License) 和 [Web Service](https://musicbrainz.org/doc/MusicBrainz_API) 规则。

- 中央规范目录只提取匹配所需的基础实体字段与 MusicBrainz 标识，并保留来源标识。
- API 候选必须标注来源为 MusicBrainz；产品发布文档提供 MusicBrainz 署名和许可链接。
- 未确认许可类别的字段按更严格的 CC BY-SA 处理，不移除来源或许可信息。
- 不缓存或再分发封面、歌词、编辑历史、用户私有数据或 API 返回的外部受限内容。
- 运营方应在发布前复核当时有效的许可与服务规则；本文是工程边界记录，不替代法律意见。

## 6. 隐私与停用

发往 MusicBrainz 的查询只包含规范化后的标题、艺术家、专辑、时长等最小元数据，不包含客户端 ID、`clientTrackId`、Token、NAS 路径、文件正文或完整原始观测。

以下任一情况应停止 Provider 调用并保留中央缓存降级服务：许可或服务规则不再允许当前用途；无法提供合规 User-Agent；持续被拒绝访问；适配器无法在响应边界内安全解析；运营方无法履行署名或数据删除要求。

## 7. 实现清单

- Provider 名称固定为 `musicbrainz`，适配器版本随解析语义变化递增。
- 允许主机固定为 `musicbrainz.org:443`，重定向不得离开该主机。
- 指标只记录 Provider、端点类别、状态分类、耗时和缓存结果，不记录完整查询或响应。
- 配置与日志不得包含客户端 Token、原始 NAS 路径或完整 Provider payload。
