# AcoustID / Chromaprint 指纹查询

Chromaprint 只在野人音乐所在 NAS 本地对音频生成；Wildman Service 不接收音频、不访问文件、不运行 Chromaprint 或 ffmpeg。客户端可在最小观测中发送已有 `fingerprint` 与 `durationMs`，中央服务据此调用 `https://api.acoustid.org/v2/lookup`。

服务端通过 `WILDMAN_ACOUSTID_API_KEY` 配置 AcoustID 应用 key，该值是秘密，不进入日志、响应或运营页面。适配器固定 HTTPS 主机、禁止重定向、每秒最多 1 请求、总超时 10 秒、响应最大 1 MiB，并尊重 429。缺少指纹或正时长时不调用 Provider。

AcoustID 返回的是识别证据而非自动写回决定。候选仍参与标题、艺术家、发行和时长评分，并与 MusicBrainz/Wikidata 来源冲突一起展示。使用、缓存与再分发需遵守当期 [AcoustID 服务条款](https://acoustid.org/terms)；指纹按隐私数据纳入客户端删除和保留策略。
