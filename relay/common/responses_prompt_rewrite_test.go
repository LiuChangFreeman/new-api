package common

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/gjson"

	"github.com/stretchr/testify/require"
)

func TestRewriteResponsesCustomPromptInstructions(t *testing.T) {
	original := []byte(`{"model":"gpt-5.5","instructions":"keep this\n\n## Intermediary updates\nremove this","max_output_tokens":1234567890123456789}`)

	rewritten, ok, reason, err := RewriteResponsesCustomPrompt(original)

	require.NoError(t, err)
	require.True(t, ok)
	require.Contains(t, reason, "instructions")
	require.NotContains(t, string(rewritten), ResponsesCustomPromptRewriteMarker)
	require.Contains(t, string(rewritten), `"max_output_tokens":1234567890123456789`)
	require.Equal(t, "keep this", gjson.GetBytes(rewritten, "instructions").String())
}

func TestRewriteResponsesCustomPromptInputDeveloperAndSystemOnly(t *testing.T) {
	original := []byte(`{
		"input":[
			{"role":"developer","content":[{"type":"input_text","text":"dev keep\n## Intermediary updates\nremove dev"}]},
			{"role":"system","content":"sys keep\n## Intermediary updates\nremove sys"},
			{"role":"user","content":[{"type":"input_text","text":"user keep\n## Intermediary updates\nkeep user"}]}
		]
	}`)

	rewritten, ok, reason, err := RewriteResponsesCustomPrompt(original)

	require.NoError(t, err)
	require.True(t, ok)
	require.Contains(t, reason, "input[0].content[0].text")
	require.Contains(t, reason, "input[1].content")
	require.Equal(t, "dev keep", gjson.GetBytes(rewritten, "input.0.content.0.text").String())
	require.Equal(t, "sys keep", gjson.GetBytes(rewritten, "input.1.content").String())
	require.Contains(t, gjson.GetBytes(rewritten, "input.2.content.0.text").String(), ResponsesCustomPromptRewriteMarker)
}

func TestRewriteResponsesCustomPromptNoMarker(t *testing.T) {
	rewritten, ok, reason, err := RewriteResponsesCustomPrompt([]byte(`{"instructions":"keep"}`))

	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, rewritten)
	require.Equal(t, "prompt rewrite marker not found", reason)
}

func TestRewriteResponsesCustomPromptPreservesUnknownFields(t *testing.T) {
	original := []byte(`{"unknown":{"nested":true},"instructions":"keep\n## Intermediary updates\nremove"}`)

	rewritten, ok, _, err := RewriteResponsesCustomPrompt(original)

	require.NoError(t, err)
	require.True(t, ok)
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rewritten, &payload))
	require.JSONEq(t, `{"nested":true}`, string(payload["unknown"]))
}
