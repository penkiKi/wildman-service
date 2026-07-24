# SQLite 备份与恢复

备份可以在服务运行时执行，使用 SQLite `VACUUM INTO` 创建一致的新文件；输出路径已存在时拒绝覆盖。

```powershell
go run ./cmd/dbtool backup -data-dir ./data -out ./backups/wildman-20260724.db
go run ./cmd/dbtool verify -file ./backups/wildman-20260724.db
```

恢复必须先停止 Wildman Service，避免进程持有旧数据库及 WAL。命令会先验证完整性和迁移版本；现有 `wildman.db` 会改名为 `.before-restore-<UTC>` 可恢复副本，然后才激活备份：

```powershell
go run ./cmd/dbtool restore -data-dir ./data -from ./backups/wildman-20260724.db -confirm
```

恢复后启动服务并检查 `/api/v1/ready`。确认无误前不要删除旧数据库副本；备份和旧副本都包含敏感曲目观测与凭证摘要，应使用受限存储并纳入保留清理。
