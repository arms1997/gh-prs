package core

import (
	"strings"
	"testing"
)

// Trimmed from a real `claude -p --output-format stream-json --verbose` run.
const (
	evInit   = `{"type":"system","subtype":"init","session_id":"s"}`
	evTool   = `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"npm test -- --watch=false","description":"Run tests"}}]}}`
	evText   = `{"type":"assistant","message":{"content":[{"type":"text","text":"Fixed the flaky assertion.\nDetails follow."}]}}`
	evResult = `{"type":"result","subtype":"success","is_error":false,"num_turns":7,"result":"Pushed abc123: fixed the test.","total_cost_usd":0.42}`
	evAuth   = `{"type":"result","subtype":"success","is_error":true,"num_turns":1,"result":"Invalid API key · Fix external API key","total_cost_usd":0}`
)

func TestJobParsing(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	v := writeJob(t, deadPID(t), strings.Join([]string{evInit, evTool, evText, evResult}, "\n")+"\n", "")
	v.Update()
	if v.Status != JobDone || v.Result != "Pushed abc123: fixed the test." || v.Turns != 7 || v.Cost != 0.42 {
		t.Fatalf("done job: %+v", v)
	}
	if want := []string{"Bash npm test -- --watch=false", "“Fixed the flaky assertion.”"}; strings.Join(v.Activity, "|") != strings.Join(want, "|") {
		t.Fatalf("activity = %q", v.Activity)
	}

	v = writeJob(t, deadPID(t), evInit+"\n"+evAuth+"\n", "")
	v.Update()
	if v.Status != JobFailed || !strings.Contains(v.Result, "Invalid API key") {
		t.Fatalf("auth failure: %+v", v)
	}

	v = writeJob(t, deadPID(t), evInit+"\n"+evTool+"\n", "warning\nError: something broke\n")
	v.Update()
	if v.Status != JobFailed || v.Result != "Error: something broke" {
		t.Fatalf("died without result: %+v", v)
	}
}
