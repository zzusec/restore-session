package agent

import "testing"

const projHome = "/Users/x"

func TestIsHomeOrEmpty(t *testing.T) {
	cases := map[string]bool{
		"":                true,
		"/Users/x":        true,
		"/Users/x/":       true,
		"/home/x":         false,
		"/Users/x/ai-crt": false,
	}
	for in, want := range cases {
		if got := IsHomeOrEmpty(in, projHome); got != want {
			t.Errorf("IsHomeOrEmpty(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestInferProjectPrefersHighestWeight(t *testing.T) {
	weights := map[string]int{
		"/Users/x/desktop-pet": 5,
		"/Users/x/ai-court":    2,
		"/Users/x/chi-shen-me": 1,
	}
	if got := InferProject(projHome, weights); got != "desktop-pet" {
		t.Errorf("InferProject = %q, want desktop-pet", got)
	}
}

func TestInferProjectExcludesHomeItself(t *testing.T) {
	weights := map[string]int{
		"/Users/x":   9,
		"/Users/x/ssh": 1,
	}
	if got := InferProject(projHome, weights); got != "ssh" {
		t.Errorf("InferProject = %q, want ssh (home itself excluded)", got)
	}
}

func TestInferProjectExcludesNonProjectDirs(t *testing.T) {
	weights := map[string]int{
		"/Users/x/.codex":       8,
		"/Users/x/Downloads":    7,
		"/Users/x/Developer":    6,
		"/Users/x/.ssh":         5,
		"/Users/x/Library":      4,
		"/Users/x/real-project": 1,
	}
	if got := InferProject(projHome, weights); got != "real-project" {
		t.Errorf("InferProject = %q, want real-project (non-project dirs excluded)", got)
	}
}

// Any config or tooling dot-directory is excluded, not just the well-known
// ones: CC Switch and a homebrew or shell-settings dir would otherwise be
// read as the project an agent happened to reconfigure.
func TestInferProjectExcludesAnyDotdir(t *testing.T) {
	weights := map[string]int{
		"/Users/x/.cc-switch":  5,
		"/Users/x/.zhengyou":   4,
		"/Users/x/desktop-pet": 1,
	}
	if got := InferProject(projHome, weights); got != "desktop-pet" {
		t.Errorf("InferProject = %q, want desktop-pet (dotdirs excluded)", got)
	}
}

func TestInferProjectEmpty(t *testing.T) {
	if got := InferProject(projHome, nil); got != "" {
		t.Errorf("InferProject(nil) = %q, want empty", got)
	}
	if got := InferProject(projHome, map[string]int{"/Users/x/.codex": 3}); got != "" {
		t.Errorf("InferProject(only blacklist) = %q, want empty", got)
	}
}

func TestInferProjectTieBreaksLexically(t *testing.T) {
	weights := map[string]int{
		"/Users/x/beta":  2,
		"/Users/x/alpha": 2,
	}
	if got := InferProject(projHome, weights); got != "alpha" {
		t.Errorf("InferProject tie = %q, want alpha", got)
	}
}

func TestInferProjectIgnoresNonHomePaths(t *testing.T) {
	weights := map[string]int{
		"/opt/other":   10,
		"/Users/x/grok": 1,
	}
	if got := InferProject(projHome, weights); got != "grok" {
		t.Errorf("InferProject = %q, want grok (path outside home excluded)", got)
	}
}

// A deep path still names the top-level project, which is how the grouping
// and the project column are meant to read in one home.
func TestInferProjectKeepsFirstSegment(t *testing.T) {
	weights := map[string]int{
		"/Users/x/chi-shen-me/server": 3,
		"/Users/x/chi-shen-me":        1,
	}
	if got := InferProject(projHome, weights); got != "chi-shen-me" {
		t.Errorf("InferProject deep = %q, want chi-shen-me", got)
	}
}