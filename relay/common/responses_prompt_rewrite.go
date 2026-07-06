package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const ResponsesCustomPromptRewriteMarker = "## Intermediary updates"

const ResponsesCustomPromptRewriteDisclosure = `删除从 "## Intermediary updates" 开始到结尾的段落`

func RewriteResponsesCustomPrompt(requestBody []byte) ([]byte, bool, string, error) {
	if len(requestBody) == 0 {
		return nil, false, "empty request body", nil
	}
	if !json.Valid(requestBody) {
		return nil, false, "request body is not valid JSON", nil
	}

	result := append([]byte(nil), requestBody...)
	rewrittenFields := make([]string, 0, 2)

	instructions := gjson.GetBytes(result, "instructions")
	if instructions.Exists() && instructions.Type == gjson.String {
		if rewritten, ok := rewriteResponsesPromptText(instructions.String()); ok {
			next, err := sjson.SetBytes(result, "instructions", rewritten)
			if err != nil {
				return nil, false, "rewrite instructions failed", err
			}
			result = next
			rewrittenFields = append(rewrittenFields, "instructions")
		}
	}

	input := gjson.GetBytes(result, "input")
	if input.Exists() && isRawJSONArray(input.Raw) {
		for itemIndex, item := range input.Array() {
			if !isResponsesPromptInstructionRole(item.Get("role").String()) {
				continue
			}
			content := item.Get("content")
			contentPath := fmt.Sprintf("input.%d.content", itemIndex)
			if content.Type == gjson.String {
				if rewritten, ok := rewriteResponsesPromptText(content.String()); ok {
					next, err := sjson.SetBytes(result, contentPath, rewritten)
					if err != nil {
						return nil, false, fmt.Sprintf("rewrite %s failed", contentPath), err
					}
					result = next
					rewrittenFields = append(rewrittenFields, fmt.Sprintf("input[%d].content", itemIndex))
				}
				continue
			}

			if !content.Exists() || !isRawJSONArray(content.Raw) {
				continue
			}
			for partIndex, part := range content.Array() {
				if !isResponsesPromptTextPart(part.Get("type").String()) {
					continue
				}
				text := part.Get("text")
				if text.Type != gjson.String {
					continue
				}
				if rewritten, ok := rewriteResponsesPromptText(text.String()); ok {
					textPath := fmt.Sprintf("input.%d.content.%d.text", itemIndex, partIndex)
					next, err := sjson.SetBytes(result, textPath, rewritten)
					if err != nil {
						return nil, false, fmt.Sprintf("rewrite %s failed", textPath), err
					}
					result = next
					rewrittenFields = append(rewrittenFields, fmt.Sprintf("input[%d].content[%d].text", itemIndex, partIndex))
				}
			}
		}
	}

	if len(rewrittenFields) == 0 {
		return nil, false, "prompt rewrite marker not found", nil
	}

	reason := fmt.Sprintf(
		"removed %q section from %d field(s): %s",
		ResponsesCustomPromptRewriteMarker,
		len(rewrittenFields),
		strings.Join(rewrittenFields, ", "),
	)
	return result, true, reason, nil
}

func rewriteResponsesPromptText(text string) (string, bool) {
	index := strings.Index(text, ResponsesCustomPromptRewriteMarker)
	if index < 0 {
		return "", false
	}
	return strings.TrimRight(text[:index], " \t\r\n"), true
}

func isRawJSONArray(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), "[")
}

func isResponsesPromptInstructionRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "developer", "system":
		return true
	default:
		return false
	}
}

func isResponsesPromptTextPart(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "", "text", "input_text":
		return true
	default:
		return false
	}
}
