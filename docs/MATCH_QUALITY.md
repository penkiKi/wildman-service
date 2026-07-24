# 匹配质量报告

## 样本边界

`docs/quality/samples.json` 只包含人工合成元数据和 `synthetic-*` 标识，不得复制生产观测、客户端 ID、NAS 路径、Token、完整 Provider payload 或可关联个人收藏的数据。新增样本在提交前必须人工复核脱敏。

## 离线报告

质量报告命令只读取指定 JSON 文件并调用领域评分函数，不访问数据库、网络或 Provider：

```powershell
go run ./cmd/quality-report docs/quality/samples.json
```

输出包含样本数、Top-1 正确数、Top-1 准确率、正确候选平均分和逐例期望/实际 Recording ID。样本集较小时只作为回归基线，不能代表真实目录总体质量。

## 扩充规则

- 使用合成、许可明确或不可逆脱敏的字段。
- 每例至少包含一个易混淆负候选，并明确唯一期望 Recording ID。
- 覆盖中文、英文、版本关键词、时长误差、多人艺术家、缺失专辑和发行曲序。
- 评分规则变化时保留旧样本，报告差异并人工审查证据与冲突，不能只追求总分。
