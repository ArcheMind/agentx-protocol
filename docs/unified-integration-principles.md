# Unified Integration Principles

本文定义 Agent X 对 Skills、Hooks 及后续跨 Agent 能力的统一原则。具体格式规范不得
扩大这些原则的范围。

## 用户目标

统一层只回答三个问题：

```text
我配置了什么？
会在哪些 Agent 生效？
没生效时原因是什么？
```

不直接帮助回答这三个问题的 Provider 内部状态，不进入公共协议。

## 标准优先

Agent X 优先采用已经存在且被用户实际使用的格式：

- Skills 使用 Agent Skills `SKILL.md`。
- Hooks 使用 Claude/Codex `hooks.json`。

只有现有格式无法表达公共能力时，才允许增加最小 Agent X 扩展。内部统一事件名、缓存
键和投影元数据不得泄漏为第二套用户 DSL。

## 单一事实源

用户只编辑 Agent X 的规范位置：

```text
~/.agents/...
<project>/.agents/...
```

Provider 文件是从规范配置生成的单向投影，不是第二份事实源：

```text
.agents source
      ↓
AgentX-managed provider artifact
```

Agent X 不从投影反向同步，不自动吸收用户的原生配置，也不覆盖不属于 Agent X 的文件或
字段。投影必须能够安全地重新生成和删除。

## 公共交集

portable profile 只包含所有目标 Agent 都能可靠实现的能力。任一目标不支持的能力不进入
公共标准，也不通过 capability flag 包装成“可选公共能力”。

超出 portable profile 的合法配置：

- 产生 warning。
- 不参与跨 Agent 适配。
- 不使同一文件中的 portable 配置失效。

原生能力继续由原生 Agent 和原生配置管理。Agent X 不实现通用 native passthrough 引擎。

## 只管理自己的状态

统一 inventory 只展示：

- Agent X 规范位置中的配置。
- Agent X 自己生成的 Provider 投影。

它不枚举任意 Claude/Codex 插件，不解析 Gemini 扩展包内部结构，也不静态分析 Pi
TypeScript 源码。原生扩展由原生 Agent 展示。

公共 inventory 不推测 `active`、`trusted` 或 runtime 可用性。Provider 的启停、信任审查
和运行时状态继续由 Provider 负责。

## 确定但不编排

对于 Hooks，user scope 和 project scope 的匹配项全部生效，project scope 不覆盖 user
scope。公共 Hook 协议不定义优先级、依赖关系或自动去重；Hook 不得依赖另一个 Hook
先执行。Skills 的同名解析由 Skills 配置协议单独定义，不从 Hook 行为类推。

## 最小展示

统一展示只包含用户可验证的信息：

```text
kind
scope
source
portable
targets
warnings
```

Hook 可以额外展示 `event` 和 `command`，Skill 可以额外展示 `name`。公共协议不要求 Hook
具有 name 或稳定的公开 ID。

## 错误边界

以下情况是错误：

- 规范文件无法解析。
- portable 必填字段缺失或类型错误。
- 投影会覆盖非 AgentX 管理的数据。

以下情况是 warning：

- 合法但超出 portable profile 的事件、handler、matcher 或字段。
- 某个目标 Agent 无法投影对应公共项。

warning 不得静默升级为公共能力，错误也不得伪装成 warning 后继续生成不完整投影。

## 实现自由

Protocol 不规定 CLI 表格样式、颜色、缓存算法、Go package 结构或临时文件布局。实现可以
自由选择这些细节，但其外部行为必须满足上述原则。
