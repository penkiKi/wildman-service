# PostgreSQL 备份与恢复

生产环境使用 PostgreSQL 官方 `pg_dump` 和 `pg_restore`。备份连接应使用具备读取全部 Wildman 表权限的专用凭证，备份文件应加密存储并纳入保留策略。

```powershell
$env:WILDMAN_DATABASE_URL="postgres://wildman:<password>@postgres:5432/wildman?sslmode=require"
pg_dump --dbname=$env:WILDMAN_DATABASE_URL --format=custom --file=./backups/wildman-20260724.dump
pg_restore --list ./backups/wildman-20260724.dump
```

恢复前停止所有 Web 与 Worker 写入，并先备份当前数据库。将目标数据库重建为空库后恢复；生产凭证、所有者和扩展由数据库运营方按环境重新配置，不从归档继承。

```powershell
pg_restore --dbname=$env:WILDMAN_DATABASE_URL --no-owner --no-privileges ./backups/wildman-20260724.dump
```

恢复后先启动一个 Web 实例，确认迁移版本和 `/api/v1/ready`，再启动其余 Web 与 Worker。定期在隔离环境演练恢复并核对关键实体数量。备份包含敏感曲目观测与凭证摘要，必须限制访问。
