package agentx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixturePaths(t *testing.T, kind string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "spec", "fixtures", kind, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no %s fixtures found", kind)
	}
	return paths
}

func TestValidFixtures(t *testing.T) {
	for _, path := range fixturePaths(t, "valid") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var session Session
			if err := json.Unmarshal(data, &session); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if err := ValidateSession(session); err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
		})
	}
}

func TestInvalidFixtures(t *testing.T) {
	for _, path := range fixturePaths(t, "invalid") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var wrapper struct {
				Reason  string          `json:"reason"`
				Session json.RawMessage `json:"session"`
			}
			if err := json.Unmarshal(data, &wrapper); err != nil {
				t.Fatalf("fixture wrapper: %v", err)
			}
			if wrapper.Reason == "" {
				t.Fatal("invalid fixture must state a reason")
			}
			var session Session
			if err := json.Unmarshal(wrapper.Session, &session); err != nil {
				return // rejected at decode time
			}
			if err := ValidateSession(session); err == nil {
				t.Fatalf("expected rejection (%s), got valid", wrapper.Reason)
			}
		})
	}
}

// Tool messages must serialize without "content"; every other role must keep
// it, even when empty, since the schema requires it there.
func TestMarshalContentPresence(t *testing.T) {
	for _, path := range fixturePaths(t, "valid") {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			t.Fatal(err)
		}
		for _, message := range session.Messages {
			out, err := json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(out, &raw); err != nil {
				t.Fatal(err)
			}
			if _, has := raw["content"]; has == (message.Role == RoleTool) {
				t.Fatalf("%s: message %s role %s: content presence wrong in %s", path, message.ID, message.Role, out)
			}
		}
	}
}
