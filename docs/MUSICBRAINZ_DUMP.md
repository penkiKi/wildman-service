# MusicBrainz Dump 评估与导入

## 结论

MusicBrainz 官方 Dump 适合作为开放基础目录，但完整关系、编辑历史、标签和派生内容不适合直接装入 Wildman 的精简目录。MVP 只导入 artist、release、recording、track 及其有序关系，不导入封面、歌词、编辑记录、用户评分或外部受限 payload。

导入前必须核对当期 [MusicBrainz 数据许可](https://musicbrainz.org/doc/About/Data_License)、Dump 校验和与发布日期。CC0 与 CC BY-SA 内容需要保持可追溯边界和署名；转换过程不能混入许可不明的数据。

## 转换契约

官方 tar/TSV 应在独立临时环境中验证校验和、解包并按依赖顺序转换为 UTF-8 JSONL：artist → release/recording → track。每行最多 1 MiB，`id` 使用 `musicbrainz:<entity>:<MBID>`，引用使用同样前缀。

```json
{"type":"artist","id":"musicbrainz:artist:<mbid>","name":"Example","sortName":"Example"}
{"type":"recording","id":"musicbrainz:recording:<mbid>","title":"Song","artistIds":["musicbrainz:artist:<mbid>"],"durationMs":180000,"isrc":""}
```

转换器必须只投影上述白名单字段，不能携带原始整行、用户数据或任意 URL。发布前保留转换脚本版本、输入 Dump 日期、SHA-256 与实体计数；JSONL 本身按来源数据许可保护和分发。

## 导入

停止 Web 与 Worker、备份 PostgreSQL 后执行：

```powershell
$env:WILDMAN_DATABASE_URL="postgres://wildman:<password>@postgres:5432/wildman?sslmode=require"
go run ./cmd/catalog-import -input ./musicbrainz-catalog.jsonl
```

导入器逐行解析并调用中央 Repository，不把文件整体载入内存；相同 ID 重复导入会更新同一实体。外键要求实体依赖顺序正确。任一行失败会停止并报告行号；此前已提交记录可通过重复导入安全续跑，但全量回滚应恢复导入前备份。

## 规模与演进

导入前在预发布环境记录数据库大小、各实体计数、导入耗时和典型查询延迟。若目标子集超出维护窗口或查询退化，缩小地区、时间或使用驱动子集；全量持续同步应使用受控导入流水线并规划独立 Worker 容量，而不是让 Web 进程承担批量写入。
