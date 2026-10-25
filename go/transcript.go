// Package transcript defines the unified transcript format shared by agentx
// and downstream consumers. The normative spec lives in
// docs/unified-transcript-format.md and schema/unified-transcript.schema.json.
package agentx

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleSystem    = "system"

	BlockText     = "text"
	BlockThinking = "thinking"
	BlockToolUse  = "tool_use"
	BlockImage    = "image"
)

type ContentBlock struct {
	Type      string `json:"type" yaml:"type"`
	Text      string `json:"text,omitempty" yaml:"text,omitempty"`
	Signature string `json:"signature,omitempty" yaml:"signature,omitempty"`
	ID        string `json:"id,omitempty" yaml:"id,omitempty"`
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Input     any    `json:"input,omitempty" yaml:"input,omitempty"`
	Source    any    `json:"source,omitempty" yaml:"source,omitempty"`
}

type Message struct {
	ID         string         `json:"id" yaml:"id"`
	ParentID   string         `json:"parentId,omitempty" yaml:"parentId,omitempty"`
	Role       string         `json:"role" yaml:"role"`
	Content    []ContentBlock `json:"content,omitempty" yaml:"content,omitempty"`
	Result     any            `json:"result,omitempty" yaml:"result,omitempty"`
	Timestamp  string         `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	RecordedAt string         `json:"recordedAt,omitempty" yaml:"recordedAt,omitempty"`
	Model      string         `json:"model,omitempty" yaml:"model,omitempty"`
	StopReason string         `json:"stopReason,omitempty" yaml:"stopReason,omitempty"`
	ToolUseID  string         `json:"toolUseId,omitempty" yaml:"toolUseId,omitempty"`
	ToolName   string         `json:"toolName,omitempty" yaml:"toolName,omitempty"`
	IsError    bool           `json:"isError,omitempty" yaml:"isError,omitempty"`
}

// MarshalJSON keeps `content` present (possibly empty) on every role except
// tool, where the schema forbids it; omitempty alone would drop empty arrays.
func (m Message) MarshalJSON() ([]byte, error) {
	type plain Message
	if m.Role == RoleTool {
		return json.Marshal(plain(m))
	}
	content := m.Content
	if content == nil {
		content = []ContentBlock{}
	}
	m.Content = nil
	return json.Marshal(struct {
		plain
		Content []ContentBlock `json:"content"`
	}{plain(m), content})
}

type Session struct {
	ID           string    `json:"id" yaml:"id"`
	Provider     string    `json:"provider" yaml:"provider"`
	IsSubagent   bool      `json:"isSubagent,omitempty" yaml:"isSubagent,omitempty"`
	Workspace    string    `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	Title        string    `json:"title,omitempty" yaml:"title,omitempty"`
	StartedAt    string    `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	UpdatedAt    string    `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	MessageCount int       `json:"messageCount" yaml:"messageCount"`
	Source       string    `json:"source,omitempty" yaml:"source,omitempty"`
	Messages     []Message `json:"messages" yaml:"messages"`
}

// ValidateSession enforces the schema's structural rules plus message-id
// uniqueness. Cross-message referential checks (parentId/toolUseId
// resolution) are intentionally not enforced; sessions may be truncated.
func ValidateSession(session Session) error {
	var errs []error
	if session.ID == "" {
		errs = append(errs, errors.New("session: id must be non-empty"))
	}
	if session.Provider == "" {
		errs = append(errs, errors.New("session: provider must be non-empty"))
	}
	if err := ValidateMessages(session.Messages); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func ValidateMessages(messages []Message) error {
	var errs []error
	seen := make(map[string]bool, len(messages))
	for i, message := range messages {
		if message.ID != "" {
			if seen[message.ID] {
				errs = append(errs, fmt.Errorf("messages[%d]: duplicate id %q", i, message.ID))
			}
			seen[message.ID] = true
		}
		if err := ValidateMessage(message); err != nil {
			errs = append(errs, fmt.Errorf("messages[%d]: %w", i, err))
		}
	}
	return errors.Join(errs...)
}

func ValidateMessage(message Message) error {
	var errs []error
	if message.ID == "" {
		errs = append(errs, errors.New("id must be non-empty"))
	}
	switch message.Role {
	case RoleUser:
		if message.Model != "" || message.StopReason != "" {
			errs = append(errs, fmt.Errorf("role %q must not carry model/stopReason", message.Role))
		}
		for i, block := range message.Content {
			if block.Type != BlockText && block.Type != BlockImage {
				errs = append(errs, fmt.Errorf("content[%d]: role %q allows only text/image blocks, got %q", i, message.Role, block.Type))
			}
		}
	case RoleSystem:
		if message.Model != "" || message.StopReason != "" {
			errs = append(errs, fmt.Errorf("role %q must not carry model/stopReason", message.Role))
		}
		for i, block := range message.Content {
			if block.Type != BlockText {
				errs = append(errs, fmt.Errorf("content[%d]: role %q allows only text blocks, got %q", i, message.Role, block.Type))
			}
		}
	case RoleTool:
		if message.Model != "" || message.StopReason != "" {
			errs = append(errs, fmt.Errorf("role %q must not carry model/stopReason", message.Role))
		}
		if len(message.Content) > 0 {
			errs = append(errs, errors.New("tool message must not carry content; use result"))
		}
	case RoleAssistant:
		for i, block := range message.Content {
			if block.Type == BlockImage {
				errs = append(errs, fmt.Errorf("content[%d]: role %q does not allow image blocks", i, message.Role))
			}
		}
	default:
		errs = append(errs, fmt.Errorf("unknown role %q", message.Role))
	}
	if message.Role != RoleTool && message.Result != nil {
		errs = append(errs, fmt.Errorf("role %q must not carry result", message.Role))
	}
	for i, block := range message.Content {
		if err := validateBlock(block); err != nil {
			errs = append(errs, fmt.Errorf("content[%d]: %w", i, err))
		}
	}
	return errors.Join(errs...)
}

func validateBlock(block ContentBlock) error {
	switch block.Type {
	case BlockText, BlockThinking:
		if block.ID != "" || block.Name != "" || block.Input != nil || block.Source != nil {
			return fmt.Errorf("%s block must not carry id/name/input/source", block.Type)
		}
		if block.Type == BlockText && block.Signature != "" {
			return errors.New("text block must not carry signature")
		}
		return nil
	case BlockImage:
		if block.Source == nil {
			return errors.New("image block requires source")
		}
		if _, ok := block.Source.(map[string]any); !ok {
			return errors.New("image source must be an object")
		}
		if block.Text != "" || block.Signature != "" || block.ID != "" || block.Name != "" || block.Input != nil {
			return errors.New("image block must not carry text/signature/id/name/input")
		}
		return nil
	case BlockToolUse:
		if block.ID == "" || block.Name == "" {
			return errors.New("tool_use block requires id and name")
		}
		if block.Text != "" || block.Signature != "" || block.Source != nil {
			return errors.New("tool_use block must not carry text/signature/source")
		}
		if block.Input != nil {
			if _, ok := block.Input.(map[string]any); !ok {
				return errors.New("tool_use input must be an object")
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown block type %q", block.Type)
	}
}
