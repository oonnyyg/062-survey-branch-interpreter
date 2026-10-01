# Survey Branch Interpreter

离线问卷分支校验与答卷解释工具。仅使用 Go 标准库，不接入网络。

## 运行

```sh
go run ./cmd/survey -input examples/survey.json -out out
```

结果写入 `<out>/results.json`。问卷定义非法时以非零状态退出，且不生成结果文件。

## 测试

```sh
go test ./...
```

## 输入格式

顶层 JSON 对象包含三个字段：

- `entry`：入口题号（字符串，不得为 `END`）。
- `questions`：题目列表，每题编号唯一。
- `responses`：答卷列表，每份答卷编号唯一。

### 题目

| 字段 | 说明 |
| --- | --- |
| `id` | 题号，唯一，不得为 `END` |
| `type` | `single`（单选）或 `integer`（整数） |
| `default_next` | 默认下一题，题号或 `END` |
| `options` | 单选题必填，选项列表 |
| `min` / `max` | 整数题必填，合法闭区间 |
| `rules` | 整数题跳转规则，互不重叠且位于合法区间内的闭区间 |

单选选项：`{"value": "yes", "next": "q2"}`，`value` 互异且非空。
整数规则：`{"min": 0, "max": 30, "next": "q3"}`，未命中任何规则时走 `default_next`。

### 答卷

```json
{"id": "r001", "answers": {"q1": "yes", "q2": 25}}
```

`answers` 按题号索引。单选答案为字符串，整数答案为 JSON 整数（布尔值、小数均非法）。

## 定义校验

全部跳转视为有向边，以下情况报错并指出位置：

- 跳转目标未知、图为环、题目从入口不可达；
- `END` 被用作题号；
- 选项重复、为空或引用不存在的目标；
- 整数规则区间重叠或超出合法范围。

## 输出格式

`results.json` 中 `results` 按答卷编号字典序排列，每项：

- `id`：答卷编号；
- `status`：`completed`（正常结束）或 `error`（答卷错误）；
- `path`：实际答题路径，每步含 `question`、`answer`、`matched`（跳转依据，如 `option "yes"`、`range [0, 30]`、`default`）与 `next`；
- `error`：仅出错时存在，记录首个问题的题号与原因（缺答、类型或范围错误）；
- `unused`：未访问题目的答案题号（字典序），不校验其值。

各答卷独立处理；调整题目或规则顺序不改变结果。

## 示例

见 `examples/survey.json`，包含正常结束、未使用答案、布尔值误作整数、缺答四种情形。
