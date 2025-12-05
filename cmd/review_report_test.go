package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	_ "embed"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
)

//go:embed testdata/report_response.json
var reportResponse []byte

func TestReviewReportCommandFiltersOutput(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &fakeReportAPI{payload: reportResponse, t: t}
	apiClientFactory = func(host string) ghcli.API {
		if host == "" {
			t.Fatalf("expected host to be resolved, got empty")
		}
		return fake
	}

	root := newRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"review", "report", "agyn/repo#51", "--reviewer", "alice", "--states", "APPROVED,COMMENTED", "--not_outdated", "--tail", "1"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	var payload struct {
		Reviews []struct {
			ID       string  `json:"id"`
			Body     *string `json:"body"`
			Comments []struct {
				ID     int `json:"id"`
				Thread []struct {
					ID int `json:"id"`
				} `json:"thread"`
			} `json:"comments"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(payload.Reviews) != 1 {
		t.Fatalf("expected 1 review in filtered output, got %d", len(payload.Reviews))
	}
	review := payload.Reviews[0]
	if review.ID != "R1" {
		t.Fatalf("expected review R1, got %s", review.ID)
	}
	if review.Body == nil {
		t.Fatalf("expected review body to be present for R1")
	}
	if len(review.Comments) != 1 {
		t.Fatalf("expected 1 comment for R1, got %d", len(review.Comments))
	}
	if len(review.Comments[0].Thread) != 1 {
		t.Fatalf("expected 1 reply after tail filter, got %d", len(review.Comments[0].Thread))
	}
	if review.Comments[0].Thread[0].ID != 303 {
		t.Fatalf("expected reply id 303, got %d", review.Comments[0].Thread[0].ID)
	}

	rawStates, ok := fake.variables["states"].([]string)
	if !ok || len(rawStates) != 2 {
		t.Fatalf("expected states variable propagated, got %#v", fake.variables["states"])
	}
}

func TestReviewReportCommandInvalidState(t *testing.T) {
	root := newRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"review", "report", "agyn/repo#51", "--states", "unknown"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !strings.Contains(err.Error(), "invalid review state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeReportAPI struct {
	t         *testing.T
	payload   []byte
	variables map[string]interface{}
}

func (f *fakeReportAPI) REST(string, string, map[string]string, interface{}, interface{}) error {
	f.t.Fatalf("unexpected REST call in report command")
	return nil
}

func (f *fakeReportAPI) GraphQL(query string, variables map[string]interface{}, result interface{}) error {
	f.variables = variables
	return json.Unmarshal(f.payload, result)
}
