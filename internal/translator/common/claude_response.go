package common

import (
	"bytes"
	"encoding/json"
)

// ClaudeNonStreamToSSE converts a native Claude Messages response into the
// equivalent data-only SSE sequence expected by the existing response
// translators. Payloads that are already streaming or are not Claude messages
// are returned unchanged.
func ClaudeNonStreamToSSE(raw []byte) []byte {
	var message struct {
		ID           string            `json:"id"`
		Type         string            `json:"type"`
		Role         string            `json:"role"`
		Model        string            `json:"model"`
		Content      []json.RawMessage `json:"content"`
		StopReason   string            `json:"stop_reason"`
		StopSequence any               `json:"stop_sequence"`
		Usage        map[string]any    `json:"usage"`
	}
	if err := json.Unmarshal(raw, &message); err != nil || message.Type != "message" {
		return raw
	}

	var out bytes.Buffer
	writeClaudeDataEvent(&out, map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id": message.ID, "type": "message", "role": message.Role, "model": message.Model,
			"content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": message.Usage,
		},
	})
	for index, rawBlock := range message.Content {
		var block map[string]any
		if err := json.Unmarshal(rawBlock, &block); err != nil {
			continue
		}
		blockType, _ := block["type"].(string)
		startBlock := cloneClaudeBlock(block)
		var delta map[string]any
		switch blockType {
		case "text":
			text, _ := block["text"].(string)
			startBlock["text"] = ""
			delta = map[string]any{"type": "text_delta", "text": text}
		case "thinking":
			thinking, _ := block["thinking"].(string)
			startBlock["thinking"] = ""
			delta = map[string]any{"type": "thinking_delta", "thinking": thinking}
		case "tool_use":
			input := block["input"]
			if input == nil {
				input = map[string]any{}
			}
			inputJSON, errMarshal := json.Marshal(input)
			if errMarshal != nil {
				inputJSON = []byte("{}")
			}
			startBlock["input"] = map[string]any{}
			delta = map[string]any{"type": "input_json_delta", "partial_json": string(inputJSON)}
		}
		writeClaudeDataEvent(&out, map[string]any{
			"type": "content_block_start", "index": index, "content_block": startBlock,
		})
		if delta != nil {
			writeClaudeDataEvent(&out, map[string]any{
				"type": "content_block_delta", "index": index, "delta": delta,
			})
		}
		writeClaudeDataEvent(&out, map[string]any{"type": "content_block_stop", "index": index})
	}
	writeClaudeDataEvent(&out, map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": message.StopReason, "stop_sequence": message.StopSequence},
		"usage": message.Usage,
	})
	writeClaudeDataEvent(&out, map[string]any{"type": "message_stop"})
	return out.Bytes()
}

func cloneClaudeBlock(block map[string]any) map[string]any {
	cloned := make(map[string]any, len(block))
	for key, value := range block {
		cloned[key] = value
	}
	return cloned
}

func writeClaudeDataEvent(out *bytes.Buffer, event any) {
	raw, errMarshal := json.Marshal(event)
	if errMarshal != nil {
		return
	}
	out.WriteString("data: ")
	out.Write(raw)
	out.WriteByte('\n')
}
