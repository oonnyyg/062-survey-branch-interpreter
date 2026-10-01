# survey-branch-interpreter

离线问卷分支校验与答卷解释工具。读取一份本地 JSON 问卷定义（入口题、题目
列表、多份答卷），校验全部跳转规则，并从入口逐题还原每份答卷的实际答题
路径。仅使用 Go 标准库，不访问网络。

## 运行

```sh
go run ./cmd/survey -input examples/survey.json -outdir examples/output
```

- `-input`：问卷 JSON 文件路径。
- `-outdir`：输出目录，不存在时自动创建，结果写入其中的 `results.json`。
- 定义不合法时以非零状态退出，并打印形如 `位置: 原因` 的错误，**不会**
  生成任何结果文件。

运行测试：

```sh
go test ./...
```

## 输入格式

顶层对象包含三个字段：

| 字段 | 说明 |
| --- | --- |
| `entry` | 入口题题号，必须存在。 |
| `questions` | 题目数组，题号全局唯一。 |
| `responses` | 答卷数组，每份答卷有唯一编号。 |

`END` 是保留的终止目标，不能作为题号。跳转目标只能是已存在的题号或
`END`。

### 单选题（`"type": "single"`）

```json
{
  "id": "q1",
  "type": "single",
  "text": "Are you a new customer?",
  "options": [
    { "id": "yes", "text": "Yes", "next": "q2" },
    { "id": "no",  "text": "No",  "next": "END" }
  ],
  "default_next": "q2"
}
```

- `options` 非空；每个选项 `id` 非空且在本题内互异。
- 答案必须是字符串，且精确匹配某个选项 `id`，然后跳到该选项的 `next`。
- 选项不得重复，也不能引用不存在的选项。

### 整数题（`"type": "integer"`）

```json
{
  "id": "q2",
  "type": "integer",
  "text": "How old are you?",
  "min": 0,
  "max": 120,
  "rules": [
    { "min": 0, "max": 17, "next": "END" }
  ],
  "default_next": "q3"
}
```

- `min`、`max` 定义合法闭区间 `[min,max]`，且 `min <= max`。
- `rules` 为按闭区间匹配的跳转规则：各区间互不重叠，且整体位于
  `[min,max]` 内。
- 答案必须是 JSON 整数（布尔值、小数、字符串均不接受），落在合法区间内；
  按区间包含关系命中第一条规则并跳转，未命中任何规则则走 `default_next`。

### 答卷

```json
{ "id": "R-001", "answers": { "q1": "yes", "q2": 15 } }
```

- `id` 非空且在所有答卷中唯一。
- `answers` 以题号为键。只有实际访问到的题目才会校验答案；未访问题目的
  答案仅列入 `unused_answers`，不校验其值。

## 校验规则

定义阶段拒绝以下情况，并指出具体位置（如 `questions[0].options[1].next`）：

- 入口为空、为 `END` 或指向不存在的题。
- 题号为空、重复或使用保留字 `END`；题目类型未知。
- 单选题缺少选项，选项 `id` 为空或重复，跳转目标为空。
- 整数题缺少/颠倒上下界，规则区间非法、越界或互相重叠。
- 任一跳转目标未知（不是题号也不是 `END`）。
- 把全部跳转视为有向边后检测到环。
- 存在从入口不可达的题。

## 答卷解释

每份答卷独立处理，从 `entry` 开始逐题前进：

- 缺答、答案类型错误、选项不存在、整数越界时立即停止，只记录首个问题，
  状态为 `"error"`，并给出 `error_question_id` 与 `error`。
- 正常走到 `END` 时状态为 `"ok"`。
- 每个路径步骤记录题号、答案、跳转依据与下一目标：
  - 依据为 `option:<选项id>`、`rule:[min,max]` 或 `default`。
- `unvisited_questions`：本份答卷未访问的所有题号。
- `unused_answers`：答案中提供但对应题目未访问的题号。

## 输出格式

答卷结果按答卷编号字典序排列，示例见 `examples/output/results.json`：

```json
{
  "entry": "q1",
  "responses": [
    {
      "response_id": "R-001",
      "status": "ok",
      "path": [
        { "question_id": "q1", "answer": "yes", "basis": "option:yes", "next": "q2" },
        { "question_id": "q2", "answer": 15, "basis": "rule:[0,17]", "next": "END" }
      ],
      "unused_answers": [],
      "unvisited_questions": ["q3", "q4"]
    }
  ]
}
```

输出只取决于题目与跳转规则的语义，调整题目数组或规则数组的排列顺序不
改变解释结果。
