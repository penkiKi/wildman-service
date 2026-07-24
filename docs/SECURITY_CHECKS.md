# 安全与供应链检查

`.github/workflows/security.yml` 在主分支、Pull Request 和手动触发时执行：

- `bun audit`：前端锁定依赖的已知漏洞。
- `govulncheck ./...`：Go 调用图相关漏洞。
- Gitleaks：完整 Git 历史中的秘密与高风险凭证模式。
- Anchore Syft：生成 `spdx-json` SBOM 并保存为 CI artifact。

任何扫描失败都必须审查后再发布；不能通过删除锁文件、排除业务目录或把真实 Token 加入允许列表规避。误报例外需要在独立变更中记录规则、具体非秘密测试值、负责人和复查日期。

依赖许可基线见 [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md)。每个版本发布前应从 SBOM 复核新增传递依赖、容器系统包、许可证兼容性及 MusicBrainz 署名。SBOM 与镜像版本/摘要一起保存。

扫描只针对源代码、锁文件和构建依赖，不读取 `/data`、备份、生产环境变量或用户观测。发现真实秘密时立即撤销/轮换，清理历史需单独评估协作影响，不能仅删除当前文件。
