# TODO List

只执行当前阶段中最靠前的未完成任务，不为未来能力提前增加抽象。

## 0. 已完成基础

- [x] `C0001` 建立 Go、React、Bun、Tailwind、Docker 和嵌入式 Web 骨架。
- [x] `C0002` 实现数据库迁移、健康/就绪接口、日志、Request ID 和统一响应信封。
- [x] `C0003` 实现运营管理员初始化、Argon2id、Session、CSRF、Origin 白名单和登录限速。
- [x] `C0004` 实现初始化、登录、退出和受保护的运营 Web 布局。
- [x] `C0005` 将产品边界调整为野人音乐本地扫描、Wildman Service 中央匹配。
- [x] `C0006` 移除服务端 `/music`、ffprobe、Chromaprint 和本地 Library 运行依赖。
- [x] `C0007` 增加客户端安装、中央目录、来源观测和解析请求数据库迁移。
- [x] `C0008` 定义 TrackObservation、Artist、Release、Recording 和 Candidate 领域模型。

## 1. Phase 1：客户端接入与观测（当前）

### 客户端凭证

- [x] `C1101` 定义 ClientInstallation、Token 格式、摘要和状态规则。
- [x] `C1102` 实现客户端凭证 Store 和运营管理员创建、列表、撤销 API。
- [x] `C1103` 实现 Bearer Token 中间件、最后使用时间和每客户端限流。
- [x] `C1104` 实现运营端客户端管理页面，完整 Token 只显示一次。

### 曲目观测与解析请求

- [x] `C1201` 定义 TrackObservation 输入限制和稳定错误码。
- [x] `C1202` 实现观测 upsert，不接收绝对路径或音频正文。
- [x] `C1203` 使用 `Idempotency-Key` 创建唯一 ResolutionRequest。
- [x] `C1204` 实现解析请求创建、查询和客户端数据隔离 API。
- [x] `C1205` 编写野人音乐客户端接入和版本兼容说明。

## 2. Phase 2：中央目录与 Provider

- [x] `C2101` 实现中央 Artist、Release、Recording、Track Repository。
- [x] `C2102` 选择首个合规 Provider 并记录认证、限流、缓存和再分发边界。
- [x] `C2103` 定义最小 Provider 接口和错误类型。
- [x] `C2104` 实现超时、响应大小、重定向白名单、限流和有限重试。
- [x] `C2105` 保存不可变 SourceObservation、payload hash、TTL 和适配器版本。
- [x] `C2106` 实现缓存命中、负缓存和相同查询并发合并。
- [x] `C2107` 建立缓存命中率、Provider 请求量和错误分类指标。

## 3. Phase 3：匹配与候选

- [x] `C3101` 实现标题、歌手、专辑、时长和版本关键词规范化。
- [x] `C3102` 实现字段评分、正向证据和冲突。
- [x] `C3103` 实现专辑发行与曲目顺序上下文。
- [x] `C3104` 持久化并返回有序 ResolutionCandidate。
- [x] `C3105` 生成字段级 Tag Patch，但不执行文件写入。
- [x] `C3106` 建立脱敏样本与匹配质量报告。

## 4. Phase 4：审核回报与发布

- [x] `C4101` 接收野人音乐的接受、拒绝和本地写回结果。
- [x] `C4102` 实现客户端、任务、Provider 和缓存运营页面。
- [x] `C4103` 实现数据保留、客户端删除和审计策略。
- [x] `C4104` 完成 PostgreSQL 备份、恢复和迁移验证。
- [x] `C4105` 完成反向代理、HTTPS、amd64/arm64 和发布文档。
- [x] `C4106` 完成安全检查、依赖许可证、秘密扫描和 SBOM。

## 5. P1：目录规模与平台能力

- [x] `C5101` 评估并导入 MusicBrainz Dump 等开放目录。
- [x] `C5102` 增加第二 Provider 和跨来源证据。
- [x] `C5103` 支持 AcoustID/Chromaprint 指纹查询。
- [x] `C5104` 切换至 PostgreSQL 和独立 Worker。
- [x] `C5105` 增加配额、设备授权、公开注册和计费。

## Definition of Done

- [ ] 每一行代码都服务于当前切片。
- [ ] Domain 不导入 HTTP、数据库或 Provider DTO。
- [ ] API 和配置变化同步文档。
- [ ] 不记录 Token、绝对 NAS 路径或完整敏感 payload。
- [ ] 所有客户端查询由认证上下文强制隔离。
- [ ] Provider 调用具备合规记录、超时、限流和响应边界。
- [ ] 数据库变化使用新的不可变迁移。
- [ ] 前后端编译和容器构建通过。
