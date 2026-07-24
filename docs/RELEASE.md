# 部署与发布

## 网络边界

Wildman Service 容器只提供 HTTP，由同机或受信任网络中的反向代理终止 HTTPS。默认 Compose 仅绑定 `127.0.0.1:8080`，不要把 8080 直接暴露到公网。生产必须配置 `WILDMAN_PROVIDER_CONTACT`，如运营联系 URL 或邮箱。

### Caddy

```caddyfile
wildman.example.com {
    encode zstd gzip
    reverse_proxy 127.0.0.1:8080
}
```

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name wildman.example.com;
    ssl_certificate /etc/letsencrypt/live/wildman.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/wildman.example.com/privkey.pem;

    client_max_body_size 1m;
    proxy_read_timeout 30s;
    proxy_send_timeout 30s;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Request-ID $request_id;
    }
}
```

只信任来自该反向代理的转发头；防火墙同时限制后端端口。TLS 最低版本、证书续期和 HSTS 由代理统一管理。运营 Web 与客户端 API 使用同一 HTTPS 域名时无需额外 CORS；跨来源运营前端必须把完整来源加入 `WILDMAN_ALLOWED_ORIGINS`。

## 数据库

Web 与 Worker 必须通过 `WILDMAN_DATABASE_URL` 连接 PostgreSQL。应用容器不持久化数据库文件，也不挂载 NAS 音乐目录。生产连接启用 TLS，并使用独立秘密管理数据库凭证。升级前按 [BACKUP_RESTORE.md](./BACKUP_RESTORE.md) 创建并验证备份；发布时先启动一个 Web 实例完成迁移，再扩容 Web 与 Worker。

## amd64 与 arm64 镜像

Dockerfile 使用纯 Go 交叉编译并支持 `linux/amd64`、`linux/arm64`。发布到镜像仓库：

```powershell
docker buildx build --platform linux/amd64,linux/arm64 -t registry.example.com/wildman-service:0.1.0 --push .
docker buildx imagetools inspect registry.example.com/wildman-service:0.1.0
```

发布使用不可变版本标签；可选 `latest` 只作为同一摘要的附加标签。镜像以 UID/GID 10001 非 root 运行，运行时只包含 CA 证书、时区数据和服务二进制。

## 发布检查

1. 构建前端、Go 服务与双架构镜像。
2. 执行依赖许可、漏洞、秘密与 SBOM 检查。
3. 验证备份和数据库迁移版本。
4. 在预发布环境通过 `/api/v1/health` 与 `/api/v1/ready` 检查启动。
5. 检查 HTTPS、请求体上限、日志脱敏和 Provider 联系信息。
