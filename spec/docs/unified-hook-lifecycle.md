# Unified Hook Lifecycle

只保留 Session 和 Turn 两级是可行的，而且四家都能稳定映射。

```text
Session
├── session.start
├── Turn *
│   ├── turn.before
│   └── turn.after
└── session.end
```

## 原生事件映射

| 公共事件 | Claude | Codex | Gemini | Pi |
|---|---|---|---|---|
| `session.start` | `SessionStart` | `SessionStart` | `SessionStart` | `session_start` |
| `session.end` | `SessionEnd` | `SessionEnd` | `SessionEnd` | `session_shutdown` |
| `turn.before` | `UserPromptSubmit` | `UserPromptSubmit` | `BeforeAgent` | `input` + `before_agent_start` |
| `turn.after` | `Stop` / `StopFailure` | `Stop` / `Interrupt` | `AfterAgent` | `agent_settled` |

Pi 的 `turn_start/turn_end` 不用于公共 Turn：它们表示一次内部模型循环，一个用户请求内可能发生多次。

## `session.start`

触发范围：

- 新建 session。
- 恢复 session。
- clear/reset 后重新初始化。
- fork/切换后的新 session；具体 source 因 Agent 而异。

公共输入：

```text
session_id
source              # startup | resume | clear | fork | other
cwd
transcript_path?
parent_session_id?
model?
agent?
```

公共能力：

- 读取 session 元数据。
- 执行初始化副作用。
- 加载项目状态。
- 注入首轮上下文。
- 输出用户可见的 warning/status。
- 初始化 Hook 自己的临时状态。

不应允许：

- 阻止 session 创建。
- 改写 session ID 或 cwd。
- 直接改写用户 prompt。
- 假装所有 Agent 都支持持久环境变量。

其中“注入首轮上下文”需要适配：

- Claude/Codex/Gemini：直接使用原生 Hook 输出。
- Pi：`session_start` 暂存上下文，在随后的 `before_agent_start` 注入。

## `session.end`

触发范围：

- 正常退出。
- clear/resume 导致旧 session 结束。
- 中断或异常关闭；能否收到事件取决于进程是否仍有清理机会。

公共输入：

```text
session_id
reason              # completed | cleared | switched | interrupted | error | other
cwd
transcript_path?
last_assistant_message?
started_at?
ended_at?
```

公共能力：

- 清理进程、文件、锁或临时资源。
- 写审计记录和统计。
- 保存 Hook 自己的状态。
- 发送完成通知。

不应允许：

- 阻止 session 结束。
- 要求一定执行成功。
- 修改 transcript。
- 触发模型继续工作。

这是 advisory/cleanup Hook，不能作为强一致事务。

## `turn.before`

定义：收到一次用户输入之后、完整 agent run 开始之前。

公共输入：

```text
session_id
turn_id?
cwd
prompt
model?
permission_mode?
```

公共能力：

- 检查 prompt。
- allow/deny 当前 Turn。
- 返回拒绝原因。
- 注入本 Turn 的附加上下文。
- 执行审计或策略检查。
- 输出用户可见 warning/status。

建议公共输出：

```text
decision            # allow | deny
reason?
additional_context?
system_message?
```

不纳入公共能力：

- 改写原始 prompt。
- 改写 system prompt。
- 更换模型。
- 修改工具集合。
- 自动批准后续所有工具调用。

原因是四家不都具备这些能力。Pi 能转换输入，Gemini/Pi 能深入修改 agent context，但 Claude/Codex 的公共安全能力主要是拒绝或附加上下文。

Pi 需要组合两个原生事件：

```text
input
  → 负责读取/拒绝用户输入

before_agent_start
  → 负责注入 additional_context
```

## `turn.after`

定义：一次用户输入对应的完整 agent run 已经得到最终结果，且没有待执行的内部模型或工具循环。

公共输入：

```text
session_id
turn_id?
outcome             # completed | denied | interrupted | error
last_assistant_message?
error?
duration_ms?
```

公共能力：

- 读取最终输出和 outcome。
- 执行审计、记录、通知。
- 验证结果是否满足外部策略。
- 返回用户可见反馈。

基础公共输出：

```text
system_message?
```

可选扩展能力：

```text
decision            # accept | continue
reason?
additional_context?
```

但 `continue` 不能列为四家共同保证：

- Claude：`Stop` 可阻止停止并继续。
- Codex：`Stop` 可要求继续。
- Gemini：`AfterAgent` 可 retry/halt。
- Pi：`agent_settled` 是最终通知，没有等价的原生返回值继续当前 run。

因此应标注：

```text
turn.after.continue:
  claude: supported
  codex: supported
  gemini: supported
  pi: unsupported
```

不能把 Pi 的主动 `sendMessage()` 包装成相同能力；那会创建新输入，而不是继续原 Turn。

## 建议的最小公共协议

```text
session.start
  observe
  side_effect
  add_context

session.end
  observe
  side_effect

turn.before
  observe
  allow_or_deny
  add_context

turn.after
  observe
  side_effect
  continue?          # capability-gated
```

最重要的限制：

- Session Hook 不参与决策，只负责初始化和清理。
- Turn Before 是统一策略入口。
- Turn After 默认只观察；继续执行属于可选能力。
- Prompt 改写、system prompt 改写、模型调用、工具调用全部留在原生扩展层，不进入统一 Hooks。
