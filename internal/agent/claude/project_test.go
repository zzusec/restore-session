package claude_test

import (
	"testing"
	"testing/fstest"

	"github.com/zzusec/restore-session/internal/agent/claude"
	"github.com/zzusec/restore-session/internal/session"
)

// spokeCwd builds a user or assistant record at a specific working directory,
// so a session's recorded cwd can be made to drift as the agent cd's.
func spokeCwd(kind, id, cwd, text string) string {
	return `{"type":"` + kind + `","sessionId":"` + id + `","cwd":"` + cwd + `",` +
		`"version":"2.1.0","entrypoint":"cli","timestamp":"2026-08-01T09:30:00Z",` +
		`"message":{"content":"` + text + `"}}`
}

func discoverProject(t *testing.T, tree fstest.MapFS) session.Session {
	t.Helper()
	found, err := claude.New("/claude", claude.WithFS(tree), claude.WithUserHome("/claude")).Discover(t.Context())
	if err != nil {
		t.Fatalf("Discover() = %v", err)
	}
	return only(t, found)
}

// A session launched from the home directory records that directory as cwd.
// When the agent then cd's into a project, the drift betrays which one it
// worked on; the highest-counted subdirectory wins as the project.
func TestProjectInferredFromCwdDrift(t *testing.T) {
	t.Parallel()

	got := discoverProject(t, fstest.MapFS{
		project + "abc.jsonl": transcript(
			spokeCwd("user", "abc", "/claude", "first"),
			spokeCwd("assistant", "abc", "/claude", "ok"),
			spokeCwd("user", "abc", "/claude", "go build"),
			spokeCwd("user", "abc", "/claude", "done?"),
			// Now the agent cd's into the project and stays there.
			spokeCwd("assistant", "abc", "/claude/desktop-pet", "cd desktop-pet"),
			spokeCwd("user", "abc", "/claude/desktop-pet", "fix the bug"),
			spokeCwd("assistant", "abc", "/claude/desktop-pet", "fixed"),
			spokeCwd("user", "abc", "/claude/desktop-pet", "ship it"),
		),
	})

	if got.Project != "desktop-pet" {
		t.Errorf("Project = %q, want desktop-pet", got.Project)
	}
	if got.Cwd != "/claude" {
		t.Errorf("Cwd = %q should stay the launch directory", got.Cwd)
	}
}

// A session that runs entirely from a project directory needs no inference:
// its recorded cwd already names the project.
func TestProjectLeftAloneWhenCwdNamesProject(t *testing.T) {
	t.Parallel()

	got := discoverProject(t, fstest.MapFS{
		project + "1.jsonl": transcript(
			spokeCwd("user", "1", "/claude/desktop-pet", "add a pet"),
		),
	})
	if got.Project != "" {
		t.Errorf("Project = %q, want empty when cwd already names the project", got.Project)
	}
}

// A session that stays in the home directory through and through has nothing
// to infer; the empty project keeps groupBy and projectOf on the recorded cwd.
func TestNoProjectWhenNoDrift(t *testing.T) {
	t.Parallel()

	got := discoverProject(t, fstest.MapFS{
		project + "1.jsonl": transcript(
			spokeCwd("user", "1", "/claude", "organising"),
			spokeCwd("assistant", "1", "/claude", "right"),
		),
	})
	if got.Project != "" {
		t.Errorf("Project = %q, want empty when every record is in home", got.Project)
	}
}

// A dotfile directory, though visited often, is not a project and must not
// label the session.
func TestProjectSkipsDotfileDirs(t *testing.T) {
	t.Parallel()

	got := discoverProject(t, fstest.MapFS{
		project + "1.jsonl": transcript(
			spokeCwd("user", "1", "/claude", "start"),
			spokeCwd("assistant", "1", "/claude/.codex", "edit config"),
			spokeCwd("user", "1", "/claude/.codex", "change key"),
			spokeCwd("assistant", "1", "/claude/.codex", "changed"),
		),
	})
	if got.Project != "" {
		t.Errorf("Project = %q, want empty when only dotfile dirs are visited", got.Project)
	}
}