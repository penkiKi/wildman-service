# 参考项目与取舍

## MusicBrainz

- 官网：https://musicbrainz.org/
- 数据下载：https://musicbrainz.org/doc/MusicBrainz_Database/Download
- 借鉴：Artist、Release、Recording、Track 分层和开放数据导入。
- 约束：导入、派生和再分发必须遵循对应数据许可；MVP 先实现按需查询，不立即导入全量 Dump。

## MusicBrainz Picard

- GitHub：https://github.com/metabrainz/picard
- 借鉴：发行版本、曲目顺序、时长和字段冲突共同参与匹配。
- 不借鉴：桌面应用、文件写入和 CD 工作流；这些由野人音乐本地负责。

## beets

- GitHub：https://github.com/beetbox/beets
- 借鉴：文件名、当前标签和指纹都是证据，候选需要可解释评分。
- 不借鉴：中央服务不管理本地目录、转码或插件脚本。

## Lidarr

- GitHub：https://github.com/Lidarr/Lidarr
- 借鉴：持久任务、失败可见、手动搜索和运营状态。
- 不借鉴：下载器、质量升级和本地媒体库管理。

## Wildman 决策

1. 野人音乐是 NAS 数据面，Wildman Service 是中央元数据控制面。
2. 两者只通过 HTTPS API 通信，不共享数据库。
3. 中央服务不访问文件，只接收最小曲目观测并返回候选或 Tag Patch。
4. 目录缓存优先，未命中才访问 Provider；不进行无差别全量抓取。
5. Provider 来源观测与中央规范实体分离，并保存时间、摘要和适配器版本。
