// Project inference shared by the agents that need it.
//
// Some coding agents record only the directory a session was launched from,
// which for a user who works inside their home directory is always the home
// directory itself. Rather than read that as the project, the agents that can
// see the conversation afterwards collect the working directories the agent
// actually ran in and hand the tallies to [InferProject], which picks the
// project those visits point at.
package agent

import (
	"path/filepath"
	"strings"
)

// nonProjectFirstSegments are home-child directories that are never a user's
// project, however often an agent lands in them. Config, caches and the
// generic macOS media folders would otherwise drown out real projects with the
// settings that happen to be opened most.
var nonProjectFirstSegments = map[string]bool{
	"Library": true, "Downloads": true, "Desktop": true, "Documents": true,
	"Developer": true, "Applications": true, "Movies": true, "Music": true,
	"Pictures": true, ".Trash": true,
}

// IsHomeOrEmpty reports whether cwd is the home directory itself or missing —
// the two cases where the recorded working directory cannot name a project.
func IsHomeOrEmpty(cwd, home string) bool {
	return cwd == "" || filepath.Clean(cwd) == filepath.Clean(home)
}

// InferProject picks the project directory a session most plausibly belonged
// to, from a tally of candidate paths gathered while reading the session.
//
// Weights are applied by the caller as the candidates are counted: a path the
// agent worked in directly can count once where a passing mention of another
// might count for less. This function only decides among what it is given.
//
// Only a strict subdirectory of home whose first segment is a real project
// competes; home itself is set aside, and so is any child the agent cd'd into
// for tooling and configuration — a name that starts with a dot, or one of the
// generic application folders. The result is the project name, the first
// segment under home (so "/Users/x/chi-shen-me/server" is "chi-shen-me"),
// which is how projects are counted and grouped in one home. Ties break to
// the lexically smallest name so the result is deterministic. A map with no
// candidate yields "".
func InferProject(home string, weights map[string]int) string {
	home = filepath.Clean(home)
	prefix := home + string(filepath.Separator)

	best, bestWeight := "", 0
	for path, weight := range weights {
		if weight <= 0 {
			continue
		}
		cleaned := filepath.Clean(path)
		if !strings.HasPrefix(cleaned, prefix) {
			continue
		}
		first := firstSegment(cleaned[len(prefix):])
		if first == "" || strings.HasPrefix(first, ".") || nonProjectFirstSegments[first] {
			continue
		}
		if weight > bestWeight || (weight == bestWeight && (best == "" || first < best)) {
			best, bestWeight = first, weight
		}
	}
	return best
}

// firstSegment is the topmost path element, which names the project home lives
// under, or "" for a path with no components at all.
func firstSegment(rel string) string {
	if i := strings.IndexByte(rel, filepath.Separator); i >= 0 {
		return rel[:i]
	}
	return rel
}