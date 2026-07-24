# 账户、设备授权、配额与计费边界

## 身份隔离

Customer Account 使用邮箱和密码公开注册，与 `users` 运营管理员完全分表。账户 Session Token 以 `wa_session_` 开头，只用于批准设备；不能调用运营 API，也不能代替 `wm_live_` 客户端 Token。所有 Token 和设备码在数据库只保存 SHA-256 摘要。

公开注册默认创建 `free` 订阅，每月 1,000 次解析额度。运营管理员可以把账户调整为 `free`/`pro`，设置非负月额度，并设置 `active`、`past_due`、`canceled` 状态。非 active 订阅不允许创建新解析任务。额度按 UTC 月和逻辑请求计数；相同客户端与 `Idempotency-Key` 重试只计一次。

## 设备授权流程

1. 野人音乐调用 `POST /api/v1/device/authorizations`，提交可识别安装名称。
2. 服务返回短期 `deviceCode`、用户输入的 `userCode`、600 秒有效期和 5 秒轮询间隔。
3. 用户注册或登录账户，用账户 Bearer Session 调用 `POST /api/v1/account/device/approve`。
4. 野人音乐按间隔调用 `POST /api/v1/device/token`。批准后完整 `wm_live_` Token 只返回一次；再次领取返回 410。

运营管理员必须先完成服务初始化，设备凭证才能落入现有安装目录。中央服务不访问设备文件，批准只建立账户与安装身份关系。

## 计费边界

当前实现的是计费控制面：方案、订阅状态、月额度、用量与运营审计。没有选择或集成支付处理商，不接收银行卡、账单地址或支付凭证，也不声称自动扣款。接入 Stripe 等支付商前必须单独确定商户主体、Webhook 签名、退款/税务、数据保留、地区与 PCI 边界；支付商只驱动订阅状态，不能直接签发客户端 Token。

运营手动变更订阅必须经过管理员 Session+CSRF，并写入 `subscription.updated` 审计。生产若需要自动收费，应把本控制面作为支付结果的下游，而不是在本服务保存卡数据。
