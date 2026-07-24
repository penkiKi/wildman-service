# PostgreSQL 与独立 Worker

设置 `WILDMAN_DATABASE_URL` 后 Web 与 Worker 使用 PostgreSQL；未设置时 Web 仍可用 SQLite，独立 Worker 明确拒绝 SQLite。生产连接串必须使用独立秘密和 TLS 参数，Compose 中的默认密码只用于本地开发，部署前必须通过 `WILDMAN_POSTGRES_PASSWORD` 替换。

```text
postgres://wildman:<password>@postgres:5432/wildman?sslmode=require
```

数据库包装器将现有 Repository 的 `?` 参数安全重绑定为 PostgreSQL `$n`，业务 SQL 保持单一实现。迁移仍按版本顺序执行；生产发布应先启动一个 Web 实例完成迁移，再扩容 Web/Worker，避免多个首次启动实例同时迁移。

`wildman-worker` 是独立进程。它使用 `FOR UPDATE SKIP LOCKED` 原子领取 `queued` 请求并切换为 `matching`，多个 Worker 不会领取同一任务。Worker 先查询中央目录；未命中时调用已配置的 MusicBrainz、Wikidata 和可选 AcoustID，保存来源观测与规范目录，再执行评分、跨来源合并、Tag Patch 和候选原子保存。Web 进程不执行匹配。Worker 将无敏感标签的 Provider/缓存指标快照写回数据库供运营页面读取。

SQLite 迁移到 PostgreSQL 时：停止写入、创建并验证 SQLite 备份、使用受控导出/导入工具迁移表数据、核对实体/请求计数及外键，再切换连接串。不能让 SQLite 与 PostgreSQL 同时接受生产写入。PostgreSQL 备份使用运营方标准 `pg_dump`/`pg_restore` 流程；SQLite `dbtool backup` 不适用于 PostgreSQL。
