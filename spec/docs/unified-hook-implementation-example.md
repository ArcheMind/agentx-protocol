# Unified Hook Implementation Example

本文是 [Unified Integration Principles](unified-integration-principles.md) 和
[Unified Hook Configuration](unified-hook-configuration.md) 派生出的非规范性实现示例。
它展示预期用户体验，不规定内部代码结构。

## 项目输入

```text
repo/
├── .agents/
│   └── hooks.json
└── hooks/
    ├── load-context.sh
    └── check-prompt.sh
```

```json
{
  "description": "Repository policy hooks",
  "hooks": {
    "SessionStart": [
      {
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
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "./hooks/check-command.sh"
          }
        ]
      }
    ]
  }
}
```

`SessionStart` 和 `UserPromptSubmit` 属于 portable profile。`PreToolUse` 是合法的
Claude/Codex 配置，但不属于 Agent X portable profile。

## 预期列表

```text
$ ax hook list
SCOPE    EVENT             COMMAND                         TARGETS
project  SessionStart      ./hooks/load-context.sh         claude,codex,gemini,pi
project  UserPromptSubmit  ./hooks/check-prompt.sh         claude,codex,gemini,pi

warning: PreToolUse is outside the portable hook profile and was not projected
```

结构化输出只包含可验证状态：

```json
{
  "hooks": [
    {
      "scope": "project",
      "source": "/repo/.agents/hooks.json",
      "event": "SessionStart",
      "command": "./hooks/load-context.sh",
      "portable": true,
      "targets": ["claude", "codex", "gemini", "pi"],
      "warnings": []
    },
    {
      "scope": "project",
      "source": "/repo/.agents/hooks.json",
      "event": "UserPromptSubmit",
      "command": "./hooks/check-prompt.sh",
      "portable": true,
      "targets": ["claude", "codex", "gemini", "pi"],
      "warnings": []
    }
  ],
  "warnings": [
    {
      "source": "/repo/.agents/hooks.json",
      "event": "PreToolUse",
      "message": "outside the portable hook profile; not projected"
    }
  ]
}
```

输出不包含推测的 `active`、`trusted`、runtime 状态、公开 Hook ID 或自动生成的 name。

## 预期投影

实现为每个 Provider 维护独立的 AgentX-owned artifact：

```text
.agents/hooks.json
        │
        ├── Claude AgentX artifact
        ├── Codex AgentX artifact
        ├── Gemini AgentX artifact
        └── Pi AgentX extension
```

这些 artifact 可以是插件、扩展或 Provider 支持的独立配置层。具体路径不是 Protocol 的
一部分，但实现必须满足：

- 不修改用户已有的原生 Hook。
- 不覆盖非 AgentX 管理的文件或字段。
- 删除 AgentX artifact 不影响用户原生配置。
- 相同规范输入重复生成相同外部行为。
- `PreToolUse` 不出现在任何跨 Agent 投影中。

如果 Provider 不能在不覆盖用户数据的情况下创建独立 artifact，该 Provider 的投影失败，
并向用户报告错误；实现不能退化为修改未知所有权的配置。

## 预期运行

`check-prompt.sh` 使用 Claude/Codex command Hook 约定：从 stdin 读取 JSON，并在拒绝时
输出：

```json
{
  "decision": "block",
  "reason": "Confirmation is required before this operation."
}
```

四个 Provider 的适配器必须使相同脚本产生相同的 portable 结果。Pi 可以在内部组合
`input` 和 `before_agent_start`，但这个实现细节不改变用户脚本或列表输出。

## 不做的事情

该实现不需要：

- 将 `PreToolUse` 自动映射为 Gemini 或 Pi 的相似事件。
- 展示或同步 `.claude`、`.codex`、`.gemini`、`.pi` 中的任意原生扩展。
- 从 Provider artifact 反向生成 `.agents/hooks.json`。
- 判断 Provider 是否已经信任、启用或实际执行某个 Hook。
- 为多个 Hook 定义优先级、依赖关系或去重规则。
