package app

import (
	"encoding/json"
	"testing"
)

// A partial settings change keeps everything it does not name (setting a harness's model
// once switched the harness off and cleared its binary).
func TestMergeJSONKeepsWhatIsNotNamed(t *testing.T) {
	cur := json.RawMessage(`{"claude":{"enabled":true,"binary":"/bin/claude","model":"","flags":["-x"]},"pi":{"enabled":false}}`)
	got := mergeJSON(cur, json.RawMessage(`{"claude":{"model":"opus","flags":null}}`))
	var m map[string]map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	c := m["claude"]
	if c["enabled"] != true || c["binary"] != "/bin/claude" || c["model"] != "opus" || c["flags"] != nil || m["pi"]["enabled"] != false {
		t.Fatalf("merged: %s", got)
	}
	if string(mergeJSON(json.RawMessage(`3`), json.RawMessage(`[1]`))) != `[1]` {
		t.Fatal("a non-object replaces")
	}
}
