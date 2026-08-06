package common

import (
	"bytes"
	"testing"
)

func TestClaudeNonStreamToSSE(t *testing.T) {
	raw := []byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-test","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"toolu_1","name":"lookup","input":{"q":"go"}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":12,"output_tokens":4}}`)

	got := ClaudeNonStreamToSSE(raw)
	for _, expected := range [][]byte{
		[]byte(`"type":"message_start"`),
		[]byte(`"text":"hello"`),
		[]byte(`"partial_json":"{\"q\":\"go\"}"`),
		[]byte(`"stop_reason":"tool_use"`),
		[]byte(`"input_tokens":12`),
		[]byte(`"output_tokens":4`),
	} {
		if !bytes.Contains(got, expected) {
			t.Fatalf("ClaudeNonStreamToSSE() = %s, missing %s", got, expected)
		}
	}
}

func TestClaudeNonStreamToSSELeavesOtherPayloadsUnchanged(t *testing.T) {
	raw := []byte(`data: {"type":"message_start"}`)
	if got := ClaudeNonStreamToSSE(raw); !bytes.Equal(got, raw) {
		t.Fatalf("ClaudeNonStreamToSSE() = %q, want unchanged %q", got, raw)
	}
}
