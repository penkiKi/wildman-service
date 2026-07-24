# 第二 Provider：Wikidata

Wikidata Query Service (`https://query.wikidata.org/sparql`) 作为第二独立来源，仅查询录音作品标签并返回 Wikidata QID。Wikidata 结构化数据按 CC0 提供；发布仍需展示来源并遵守当期 [Wikidata 数据访问与机器人规则](https://www.wikidata.org/wiki/Wikidata:Data_access)。

适配器要求可联系的 User-Agent，固定 HTTPS 主机，单实例每秒最多 1 请求，总超时 10 秒、响应最大 1 MiB、最多一次同主机重定向。429 返回上层退避，不通过代理或多身份规避。查询只包含规范化后的最小标题，不发送客户端身份、路径或原始观测。

Wikidata 标签结果不能单独覆盖 MusicBrainz/中央目录。跨来源证据只在 ISRC 相同，或规范化标题与完整艺术家集合相同时合并；至少两个不同来源才增加固定 0.05 分和“多个独立来源一致”证据。同一来源重复记录不加分，冲突继续保留给用户审核。

Wikidata 不提供本项目所需的稳定专辑/曲序覆盖时，相关字段保持缺失，不从描述文本猜测。缓存与 SourceObservation 规则沿用通用 Provider 边界。
