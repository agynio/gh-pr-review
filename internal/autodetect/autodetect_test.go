package autodetect

import (
	"testing"
)

func TestContext(t *testing.T) {
	// Basic struct test
	ctx := Context{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 42,
	}

	if ctx.Owner != "testowner" {
		t.Errorf("expected owner testowner, got %s", ctx.Owner)
	}
	if ctx.Repo != "testrepo" {
		t.Errorf("expected repo testrepo, got %s", ctx.Repo)
	}
	if ctx.Number != 42 {
		t.Errorf("expected number 42, got %d", ctx.Number)
	}
}

func TestDetectionError(t *testing.T) {
	err := &DetectionError{
		Operation: "test operation",
		Message:   "test message",
	}

	expected := "auto-detect test operation: test message"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

// Note: Testing Detect() requires mocking gh CLI or running in a real git repo.
// Integration tests will cover the actual detection logic.
