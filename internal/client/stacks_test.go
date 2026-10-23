package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateStackEnvSendsBackAutoUpdate(t *testing.T) {
	const autoUpdate = `{"Interval":"","Webhook":"0b9d4d4e-5a1c-4f2b-9e3a-6c1d2f3e4a5b","ForceUpdate":false,"ForcePullImage":false}`
	var sent map[string]json.RawMessage

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/stacks/7":
			io.WriteString(w, `{"Id":7,"Name":"healthchecks","EndpointId":2,
				"Env":[{"name":"DATA_DIR","value":"/data/hc"}],
				"GitConfig":{"ReferenceName":"refs/heads/main","TLSSkipVerify":true},
				"AutoUpdate":`+autoUpdate+`}`)
		case r.Method == "POST" && r.URL.Path == "/api/stacks/7/git":
			if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
				t.Errorf("decode payload: %v", err)
			}
			io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "key")
	detail, err := c.GetStack(7)
	if err != nil {
		t.Fatal(err)
	}
	payload := UpdateStackPayload{
		Env:                     append(detail.Env, EnvVar{Name: "DOMAIN", Value: "nojatrade.io"}),
		RepositoryReferenceName: detail.GitConfig.ReferenceName,
		TLSSkipVerify:           detail.GitConfig.TLSSkipVerify,
		AutoUpdate:              detail.AutoUpdate,
	}
	if err := c.UpdateStackEnv(detail.ID, detail.EndpointID, payload); err != nil {
		t.Fatal(err)
	}

	if got := string(sent["autoUpdate"]); got != autoUpdate {
		t.Errorf("autoUpdate sent = %s, want %s", got, autoUpdate)
	}
	if got := string(sent["tlsSkipVerify"]); got != "true" {
		t.Errorf("tlsSkipVerify sent = %s, want true", got)
	}
}

func TestUpdateStackPayloadWithoutAutoUpdate(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage("null")} {
		body, err := json.Marshal(UpdateStackPayload{AutoUpdate: raw})
		if err != nil {
			t.Fatal(err)
		}
		var sent map[string]json.RawMessage
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatal(err)
		}
		if v, ok := sent["autoUpdate"]; ok && string(v) != "null" {
			t.Errorf("AutoUpdate %q sent as %s", raw, v)
		}
	}
}
