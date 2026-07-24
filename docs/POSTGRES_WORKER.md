# PostgreSQL 与独立 Worker

Web 与 Worker 只支持 PostgreSQL，并且都要求设置 `WILDMAN_DATABASE_URL`；缺失时进程立即退出。生产连接串必须使用独立秘密和 TLS 参数，Compose 中的默认密码只用于本地开发，部署前必须通过 `WILDMAN_POSTGRES_PASSWORD` 替换。

```text
postgres://wildman:<password>@postgres:5432/wildman?sslmode=require
```

数据库包装器将现有 Repository 的 `?` 参数安全重绑定为 PostgreSQL `$n`，业务 SQL 保持单一实现。迁移仍按版本顺序执行；生产发布应先启动一个 Web 实例完成迁移，再扩容 Web/Worker，避免多个首次启动实例同时迁移。

`wildman-worker` 是独立进程。它使用 `FOR UPDATE SKIP LOCKED` 原子领取 `queued` 请求并切换为 `matching`，多个 Worker 不会领取同一任务。Worker 先查询中央目录；未命中时调用已配置的 MusicBrainz、Wikidata 和可选 AcoustID，保存来源观测与规范目录，再执行评分、跨来源合并、Tag Patch 和候选原子保存。Web 进程不执行匹配。Worker 将无敏感标签的 Provider/缓存指标快照写回数据库供运营页面读取。

PostgreSQL 备份和恢复使用运营方标准 `pg_dump`/`pg_restore` 流程，详见 [BACKUP_RESTORE.md](./BACKUP_RESTORE.md)。
