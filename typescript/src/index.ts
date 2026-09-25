// Unified transcript format shared by agentx and downstream consumers.
// Normative spec: docs/unified-transcript-format.md and
// schema/unified-transcript.schema.json in this repository.

export interface TextBlock {
  type: "text";
  text: string;
}

export interface ThinkingBlock {
  type: "thinking";
  text?: string;
  signature?: string;
}

export interface ToolUseBlock {
  type: "tool_use";
  id: string;
  name: string;
  input?: Record<string, unknown>;
}

export type ContentBlock = TextBlock | ThinkingBlock | ToolUseBlock;

interface MessageBase {
  id: string;
  parentId?: string | null;
  timestamp?: string;
}

export interface UserMessage extends MessageBase {
  role: "user";
  content: TextBlock[];
}

export interface AssistantMessage extends MessageBase {
  role: "assistant";
  content: ContentBlock[];
  model?: string;
  stopReason?: string;
}

export interface ToolMessage extends MessageBase {
  role: "tool";
  content: TextBlock[];
  toolUseId: string;
  toolName: string;
  isError?: boolean;
}

export type Message = UserMessage | AssistantMessage | ToolMessage;

export interface Session {
  id: string;
  provider: string;
  isSubagent?: boolean;
  workspace?: string;
  title?: string;
  startedAt?: string;
  updatedAt?: string;
  messageCount?: number;
  source?: string;
  messages: Message[];
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const isNonEmptyString = (value: unknown): value is string =>
  typeof value === "string" && value.length > 0;

// Validators return a list of problems (empty = valid) and enforce the
// schema's structural rules plus message-id uniqueness. Cross-message
// referential checks (parentId/toolUseId resolution) are intentionally not
// enforced; sessions may be truncated.

export function validateSession(value: unknown): string[] {
  if (!isRecord(value)) return ["session: must be an object"];
  const problems: string[] = [];
  if (!isNonEmptyString(value.id)) problems.push("session: id must be a non-empty string");
  if (!isNonEmptyString(value.provider)) problems.push("session: provider must be a non-empty string");
  if (!Array.isArray(value.messages)) {
    problems.push("session: messages must be an array");
  } else {
    problems.push(...validateMessages(value.messages));
  }
  return problems;
}

export function validateMessages(value: unknown): string[] {
  if (!Array.isArray(value)) return ["messages: must be an array"];
  const problems: string[] = [];
  const seen = new Set<string>();
  value.forEach((message, index) => {
    if (isRecord(message) && isNonEmptyString(message.id)) {
      if (seen.has(message.id)) problems.push(`messages[${index}]: duplicate id ${JSON.stringify(message.id)}`);
      seen.add(message.id);
    }
    problems.push(...validateMessage(message).map((problem) => `messages[${index}]: ${problem}`));
  });
  return problems;
}

export function validateMessage(value: unknown): string[] {
  if (!isRecord(value)) return ["must be an object"];
  const problems: string[] = [];
  if (!isNonEmptyString(value.id)) problems.push("id must be a non-empty string");
  if (value.parentId !== undefined && value.parentId !== null && typeof value.parentId !== "string") {
    problems.push("parentId must be a string or null");
  }
  if (value.timestamp !== undefined && typeof value.timestamp !== "string") {
    problems.push("timestamp must be a string");
  }
  const role = value.role;
  if (role !== "user" && role !== "assistant" && role !== "tool") {
    problems.push(`unknown role ${JSON.stringify(role)}`);
    return problems;
  }
  const content = value.content;
  if (!Array.isArray(content)) {
    problems.push("content must be an array");
  } else {
    content.forEach((block, index) => {
      problems.push(...validateBlock(block, role).map((problem) => `content[${index}]: ${problem}`));
    });
  }
  if (role === "assistant") {
    if (value.model !== undefined && typeof value.model !== "string") problems.push("model must be a string");
    if (value.stopReason !== undefined && typeof value.stopReason !== "string") problems.push("stopReason must be a string");
  } else if (value.model !== undefined || value.stopReason !== undefined) {
    problems.push(`role "${role}" must not carry model/stopReason`);
  }
  if (role === "tool") {
    if (!isNonEmptyString(value.toolUseId)) problems.push("tool message requires toolUseId");
    if (!isNonEmptyString(value.toolName)) problems.push("tool message requires toolName");
    if (value.isError !== undefined && typeof value.isError !== "boolean") problems.push("isError must be a boolean");
  } else {
    if (value.toolUseId !== undefined || value.toolName !== undefined) {
      problems.push(`role "${role}" must not carry toolUseId/toolName`);
    }
    if (value.isError !== undefined) problems.push(`role "${role}" must not carry isError`);
  }
  return problems;
}

function validateBlock(value: unknown, role: "user" | "assistant" | "tool"): string[] {
  if (!isRecord(value)) return ["must be an object"];
  const problems: string[] = [];
  switch (value.type) {
    case "text":
      if (typeof value.text !== "string") problems.push("text block requires text string");
      forbidKeys(value, ["type", "text"], problems);
      break;
    case "thinking":
      if (role !== "assistant") problems.push(`role "${role}" allows only text blocks, got "thinking"`);
      if (value.text !== undefined && typeof value.text !== "string") problems.push("thinking text must be a string");
      if (value.signature !== undefined && typeof value.signature !== "string") problems.push("signature must be a string");
      forbidKeys(value, ["type", "text", "signature"], problems);
      break;
    case "tool_use":
      if (role !== "assistant") problems.push(`role "${role}" allows only text blocks, got "tool_use"`);
      if (!isNonEmptyString(value.id)) problems.push("tool_use block requires id");
      if (!isNonEmptyString(value.name)) problems.push("tool_use block requires name");
      if (value.input !== undefined && !isRecord(value.input)) problems.push("tool_use input must be an object");
      forbidKeys(value, ["type", "id", "name", "input"], problems);
      break;
    default:
      problems.push(`unknown block type ${JSON.stringify(value.type)}`);
  }
  return problems;
}

function forbidKeys(value: Record<string, unknown>, allowed: string[], problems: string[]): void {
  for (const key of Object.keys(value)) {
    if (!allowed.includes(key)) problems.push(`unexpected key ${JSON.stringify(key)}`);
  }
}

export const isTextBlock = (block: ContentBlock): block is TextBlock => block.type === "text";
export const isThinkingBlock = (block: ContentBlock): block is ThinkingBlock => block.type === "thinking";
export const isToolUseBlock = (block: ContentBlock): block is ToolUseBlock => block.type === "tool_use";

export const textContent = (message: Message): string =>
  (message.content as ContentBlock[])
    .filter(isTextBlock)
    .map((block) => block.text)
    .join("\n");
