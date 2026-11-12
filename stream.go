package main

// Parsing Claude's stream-json output into a job's status and activity.

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type streamEvent struct {
	Type    string  `json:"type"`
	Subtype string  `json:"subtype"`
	IsError bool    `json:"is_error"`
	Result  string  `json:"result"`
	Cost    float64 `json:"total_cost_usd"`
	Turns   int     `json:"num_turns"`
	Message struct {
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Text  string          `json:"text"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

func (v *jobView) apply(line []byte) {
	var ev streamEvent
	if len(bytes.TrimSpace(line)) == 0 || json.Unmarshal(line, &ev) != nil {
		return
	}
	switch ev.Type {
	case "assistant":
		if v.Status != JobRunning { // a /loop job starting its next round
			v.Status, v.Finished = JobRunning, time.Time{}
		}
		for _, c := range ev.Message.Content {
			switch c.Type {
			case "tool_use":
				v.addActivity(strings.TrimSpace(c.Name + " " + summarizeInput(c.Input)))
			case "text":
				v.addActivity("“" + firstLine(c.Text) + "”")
			}
		}
	case "result":
		v.Status = JobDone
		if ev.IsError || ev.Subtype != "success" {
			v.Status = JobFailed
		}
		v.Result = strings.TrimSpace(sanitize(ev.Result))
		if v.Result == "" && v.Status == JobFailed {
			v.Result = ev.Subtype
		}
		v.Cost, v.Turns = ev.Cost, ev.Turns
	}
}

func (v *jobView) addActivity(s string) {
	if s = sanitize(s); s == "" {
		return
	}
	v.Activity = append(v.Activity, s)
	if len(v.Activity) > maxActivity {
		v.Activity = v.Activity[len(v.Activity)-maxActivity:]
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

// summarizeInput picks the most telling field of a tool call: the command, the file, the pattern.
func summarizeInput(raw json.RawMessage) string {
	var in map[string]any
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	for _, k := range []string{"command", "file_path", "pattern", "url", "description", "prompt"} {
		if s, ok := in[k].(string); ok && s != "" {
			return firstLine(s)
		}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}
