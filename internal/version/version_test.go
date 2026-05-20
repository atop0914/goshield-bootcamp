package version

import (
	"strings"
	"testing"
)

func TestGet_DefaultValues(t *testing.T) {
	info := Get()
	if info.Version != "dev" {
		t.Errorf("expected Version='dev', got %q", info.Version)
	}
	if info.GitCommit != "unknown" {
		t.Errorf("expected GitCommit='unknown', got %q", info.GitCommit)
	}
	if info.BuildDate != "unknown" {
		t.Errorf("expected BuildDate='unknown', got %q", info.BuildDate)
	}
}

func TestGet_CustomValues(t *testing.T) {
	// Save originals
	origVersion := Version
	origCommit := GitCommit
	origDate := BuildDate
	defer func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	}()

	Version = "v1.2.3"
	GitCommit = "abc1234"
	BuildDate = "2026-05-19T20:33:00Z"

	info := Get()
	if info.Version != "v1.2.3" {
		t.Errorf("expected Version='v1.2.3', got %q", info.Version)
	}
	if info.GitCommit != "abc1234" {
		t.Errorf("expected GitCommit='abc1234', got %q", info.GitCommit)
	}
	if info.BuildDate != "2026-05-19T20:33:00Z" {
		t.Errorf("expected BuildDate='2026-05-19T20:33:00Z', got %q", info.BuildDate)
	}
}

func TestInfo_String(t *testing.T) {
	info := Info{
		Version:   "v1.0.0",
		GitCommit: "deadbeef",
		BuildDate: "2026-01-01",
	}

	s := info.String()

	if !strings.Contains(s, "goshield") {
		t.Errorf("expected string to contain 'goshield', got %q", s)
	}
	if !strings.Contains(s, "v1.0.0") {
		t.Errorf("expected string to contain 'v1.0.0', got %q", s)
	}
	if !strings.Contains(s, "deadbeef") {
		t.Errorf("expected string to contain 'deadbeef', got %q", s)
	}
	if !strings.Contains(s, "2026-01-01") {
		t.Errorf("expected string to contain '2026-01-01', got %q", s)
	}
}

func TestInfo_StringFormat(t *testing.T) {
	info := Info{
		Version:   "v2.0.0",
		GitCommit: "abc1234",
		BuildDate: "2026-05-19",
	}

	s := info.String()
	expected := "goshield v2.0.0 (commit: abc1234, built: 2026-05-19)"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestInfo_StringEmptyValues(t *testing.T) {
	info := Info{}
	s := info.String()
	if !strings.Contains(s, "goshield") {
		t.Errorf("expected 'goshield' in empty info string, got %q", s)
	}
}

func TestVersionVarsAreSettable(t *testing.T) {
	// Verify that the ldflags variables are actually settable (package-level vars, not consts)
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = "test-version"
	if Version != "test-version" {
		t.Error("expected Version to be settable")
	}
}
