# Unified Hook Configuration

本文定义 Agent X Protocol 的用户侧 Hook 配置。生命周期和能力边界由
[Unified Hook Lifecycle](unified-hook-lifecycle.md) 定义。

## 设计原则

Agent X 不发明新的 Hook DSL。用户配置采用 Claude Code 与 Codex 已使用的
`hooks.json` 结构；Agent X 只定义其中能够跨 Claude、Codex、Gemini 和 Pi
运行的公共 profile。

配置语法与公共能力是两层：

- 语法层接受 Claude/Codex `hooks.json`。
- 公共能力层只保证四个生命周期事件和 command handler。
- 超出公共 profile 的合法原生配置产生 warning，不使整个配置失败。
- Agent X 内部的 `session.start` 等名称不出现在用户配置中。

## 发现位置

规范位置为：

```text
~/.agents/hooks.json          # user scope
<project>/.agents/hooks.json  # project scope
```

两个文件同时存在时，其匹配 Hook 全部生效；项目文件不会覆盖用户文件。Hook
之间不得依赖执行顺序。

工具也可以显式读取已有的 Claude/Codex `hooks.json`。其内容不需要改写成另一种
格式。

## 配置结构

```json
{
  "description": "Optional description",
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {
            "type": "command",
            "command": "./hooks/load-context.sh",
            "timeout": 10
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "./hooks/check-prompt.sh",
            "timeout": 5
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "./hooks/report.sh",
            "timeout": 10
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "./hooks/cleanup.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

### 公共事件

| 用户配置事件 | Agent X Protocol 事件 |
|---|---|
| `SessionStart` | `session.start` |
| `SessionEnd` | `session.end` |
| `UserPromptSubmit` | `turn.before` |
| `Stop` | `turn.after` |

事件名区分大小写。

### Matcher group

每个事件包含若干 matcher group：

```json
{
  "matcher": "startup|resume",
  "hooks": []
}
```

- `hooks` 必须是非空数组。
- `matcher` 可省略；省略、空字符串和 `*` 都表示全部匹配。
- `SessionStart` 的 matcher 匹配公共 `source`。
- `SessionEnd` 的 matcher 匹配公共 `reason`。
- `UserPromptSubmit` 和 `Stop` 不提供公共 matcher subject；其 `matcher` 被忽略并产生 warning。
- 为保证跨运行时一致，portable matcher 应只使用字面值、`|`、`^`、`$` 和 `.*`。
  更复杂但原生合法的正则可以保留，但属于 native-only 配置。

### Command handler

公共 handler 为：

```json
{
  "type": "command",
  "command": "./hooks/check.sh",
  "timeout": 10
}
```

| 字段 | 要求 | 含义 |
|---|---|---|
| `type` | 必须为 `command` | 执行外部命令 |
| `command` | 必填非空字符串 | 与 Claude/Codex 相同的 shell command |
| `timeout` | 可选正数 | 秒；从进程启动到退出的总时限 |

命令以 session 的 `cwd` 作为工作目录。stdin 是一个 JSON 对象；stdout 只能是空输出、
纯文本或一个 JSON 对象。诊断日志必须写入 stderr。

同一事件的多个匹配 handler 可能并发执行。配置不得依赖声明顺序，也不得依赖另一个
handler 先完成。

## 命令输入

公共输入保持 Claude/Codex 的 snake_case 形态。

所有事件：

```text
session_id
cwd
hook_event_name
transcript_path?
```

事件字段：

```text
SessionStart
  source                    # startup | resume | clear | fork | other
  parent_session_id?
  model?
  agent?

SessionEnd
  reason                    # completed | cleared | switched | interrupted | error | other
  last_assistant_message?
  started_at?
  ended_at?

UserPromptSubmit
  turn_id?
  prompt
  model?
  permission_mode?

Stop
  turn_id?
  outcome                   # completed | denied | interrupted | error
  last_assistant_message?
  error?
  duration_ms?
```

Provider 可以附加字段。portable Hook 必须忽略不认识的输入字段。

## 命令输出

用户侧输出继续采用 Claude/Codex 形态；Agent X 在内部转换为统一决策。

### `SessionStart`

```json
{
  "systemMessage": "Workspace policy loaded.",
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Read CONTRIBUTING.md before editing."
  }
}
```

`SessionStart` 不能阻止 session 创建。

### `SessionEnd`

```json
{
  "systemMessage": "Session audit saved."
}
```

输出不能阻止 session 结束，也不能修改 transcript。

### `UserPromptSubmit`

允许且不添加上下文时，退出 `0` 且不输出内容。允许并添加上下文时：

```json
{
  "systemMessage": "Repository policy applied.",
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "Do not modify generated files."
  }
}
```

拒绝当前 Turn 时：

```json
{
  "decision": "block",
  "reason": "Confirmation is required before this operation."
}
```

退出码 `2` 且将原因写入 stderr 是等价的拒绝形式。其他非零退出码表示 Hook 失败，
不能隐式拒绝 Turn。

### `Stop`

```json
{
  "systemMessage": "Turn audit saved."
}
```

公共 `Stop` 是观察与副作用 Hook。`decision: "block"`、`continue`、重试和自动创建
新输入不属于公共协议。

## 超出公共 Profile

格式正确但超出公共 profile 的内容不得使整个配置加载失败。

| 情况 | 统一模式行为 |
|---|---|
| 未知或非公共事件 | warning；不跨 Agent 执行 |
| 非 `command` handler | warning；不跨 Agent 执行 |
| 复杂 matcher | warning；只在明确支持它的原生目标透传 |
| 额外的纯元数据字段 | warning；公共适配器忽略 |
| 改变执行语义的字段，例如 `async` | warning；只在明确支持它的原生目标透传，其他目标跳过该 handler |
| 公共事件中的非公共输出能力 | warning；忽略该能力，不影响同一输出中的公共字段 |

当目标 Agent 原生支持超出项时，Agent X 可以原样透传；这属于 native-only 行为，不得
被报告为跨 Agent 能力。无法透传时只跳过对应事件或 handler，不跳过同一文件中的其他
公共 Hook。

无效 JSON、缺少 `hooks`、空 `command`、非正数 `timeout` 等不是“超出公共 profile”，
而是无效配置，必须报错。

## Provider 适配

- Claude：使用原生事件名、配置结构和 command I/O。
- Codex：使用原生事件名、配置结构和 command I/O。
- Gemini：将四个公共事件映射到 `SessionStart`、`SessionEnd`、`BeforeAgent`、`AfterAgent`。
- Pi：使用公开 ExtensionAPI；`input` 负责 `UserPromptSubmit` 的拒绝，
  `before_agent_start` 注入其 `additionalContext`，`agent_settled` 实现公共 `Stop`。

适配器不得用一个新 Turn 模拟当前 Turn 的继续执行，也不得把原生扩展能力伪装成公共能力。
