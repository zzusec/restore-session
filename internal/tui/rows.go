package tui

import (
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/zzusec/restore-session/internal/i18n"
	"github.com/zzusec/restore-session/internal/session"
	"github.com/zzusec/restore-session/internal/tui/text"
	"github.com/zzusec/restore-session/internal/tui/theme"
)

// Column widths, in cells. The date and project columns size themselves to the
// data; the date only grows when sessions from another year are in view.
const (
	dayMinWidth     = 5
	projectMinWidth = 8
	projectMaxWidth = 18
	clockWidth      = 5
	sizeWidth       = 6
)

func (m *Model) banner() string {
	var line text.Line

	// A mode shows in the pill and in the words after the counts. The bar
	// itself stays the colour every other bar is: a whole line of red fights
	// the list under it for attention it does not need to win.
	pill := m.theme.Pill
	switch {
	case m.danger:
		pill = m.theme.PillDanger
	case !m.picked.Empty():
		pill = m.theme.PillSelect
	}
	line.Fill(m.theme.Banner)
	line.Add(" "+m.meta.Label+" ", pill).Space(2)

	counts := []string{
		m.print.N(i18n.BannerSessionsOne, i18n.BannerSessionsMany, m.forest.Len()),
	}
	// Only counts worth showing: the banner is the one line that stays put, so
	// a zero would be noise.
	if archived := m.countIf(func(s session.Session) bool { return s.Archived }); archived > 0 {
		counts = append(counts,
			m.print.N(i18n.BannerArchivedOne, i18n.BannerArchivedMany, archived))
	}
	if stranded := m.forest.Orphans(); m.meta.OrphanLabel != 0 && len(stranded) > 0 {
		counts = append(counts, m.print.N(
			i18n.BannerOrphansOne, i18n.BannerOrphansMany, len(stranded),
			i18n.Args{"what": m.print.Label(m.meta.OrphanLabel, len(stranded))},
		))
	}
	line.Add(strings.Join(counts, " · "), m.theme.Muted)

	// The project view is how the list is arranged rather than what the keys
	// are about, so it sits with the counts rather than with the modes.
	if m.view == viewProject {
		line.Space(3).Add(m.print.T(i18n.BannerByProject), m.theme.Muted)
	}

	// A mode changes what the keys do, which has to be impossible to miss.
	switch {
	case m.danger:
		line.Space(3).Add(m.print.T(i18n.BannerDanger), m.theme.ModeDanger)
	case !m.picked.Empty():
		line.Space(3).Add(m.print.N(
			i18n.BannerSelectedOne, i18n.BannerSelectedMany, m.picked.Len(),
		), m.theme.Mode)
	}

	return line.Render(m.width)
}

func (m *Model) countIf(keep func(session.Session) bool) int {
	found := 0
	for _, row := range m.rows {
		if keep(row.Session) {
			found++
		}
	}
	return found
}

func (m *Model) listLines(width, height int) []string {
	if len(m.rows) == 0 {
		return []string{m.theme.Placeholder.Render(m.print.T(i18n.NoSessionsHere))}
	}

	// The leftmost column is a group label either way: the day in the timeline
	// view, the project in the project view. Whichever one is shown sizes the
	// column, since only one is ever drawn.
	today := time.Now()
	var labels []string
	if m.view == viewProject {
		labels = m.projectHeaders()
	} else {
		labels = m.dayLabels(today)
	}
	dayWidth := dayMinWidth
	for _, label := range labels {
		dayWidth = max(dayWidth, text.Width(label))
	}
	projectWidth := projectMinWidth
	for _, row := range m.rows {
		if row.Nested {
			continue
		}
		projectWidth = max(projectWidth, text.Width(projectOf(row.Session)))
	}
	projectWidth = min(projectWidth, projectMaxWidth)

	lines := make([]string, 0, height)
	for i := m.top; i < len(m.rows) && len(lines) < height; i++ {
		lines = append(lines, m.rowLine(i, labels[i], dayWidth, projectWidth, width))
	}
	return lines
}

// dayLabels names the day on the first row of each one, so a run of sessions
// reads as a group rather than spending a column on the same string 20 times.
//
// Nested rows are skipped on both counts. A sub-agent belongs to the
// conversation above it, not to a day of its own, and letting one break the run
// would make the next top-level session repeat a date that never changed.
func (m *Model) dayLabels(today time.Time) []string {
	labels := make([]string, len(m.rows))
	previous := ""
	for i, row := range m.rows {
		if row.Nested {
			continue
		}
		when := row.Session.RecencyAt()
		day := when.Format("2006-01-02")
		if day != previous {
			labels[i] = dayLabel(m.print, when, today)
		}
		previous = day
	}
	return labels
}

// projectHeaders names the project on the first top-level row of each group,
// the same way dayLabels names the day. The full working directory is the key,
// so two projects that happen to share a basename are not run together.
//
// Only the top-level rows of a group carry the label: a sub-agent belongs to
// the conversation above it and shares its directory, so repeating it would
// only bury the tree.
func (m *Model) projectHeaders() []string {
	headers := make([]string, len(m.rows))
	previous := ""
	for i, row := range m.rows {
		if row.Nested {
			continue
		}
		key := effectiveProjectKey(row.Session)
		if key != previous {
			headers[i] = projectOf(row.Session)
		}
		previous = key
	}
	return headers
}

func dayLabel(print *i18n.Printer, when, today time.Time) string {
	switch days := daysApart(when, today); {
	case days == 0:
		return print.T(i18n.Today)
	case days == 1:
		return print.T(i18n.Yesterday)
	case when.Year() != today.Year():
		// Include the year so an old session cannot look recent.
		return when.Format("06-01-02")
	default:
		return when.Format("01-02")
	}
}

// daysApart counts calendar days between two moments.
//
// Both are pinned to midday before subtracting: a day that a daylight-saving
// change makes 23 hours long would otherwise round down to zero, and yesterday
// would be labelled "Today" twice a year.
func daysApart(when, today time.Time) int {
	noon := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, t.Location())
	}
	return int(math.Round(noon(today).Sub(noon(when)).Hours() / 24))
}

func (m *Model) rowLine(index int, label string, dayWidth, projectWidth, width int) string {
	row := m.rows[index]
	s := row.Session
	cursor, picked := index == m.cursor, m.picked.Has(s.ID)

	var line text.Line
	line.Fill(m.rowFill(cursor, picked))

	// Two gutter cells repeating what the fill says: where the cursor is, and
	// what has been picked out.
	if cursor {
		line.Add(theme.CursorMark, m.theme.Cursor)
	} else {
		line.Add(theme.Blank, m.theme.Cursor)
	}
	if picked {
		line.Add(theme.PickedMark, m.theme.Picked)
	} else {
		line.Add(theme.Blank, m.theme.Picked)
	}

	// Archived rows were set aside deliberately; orphaned ones were left
	// behind by accident. One colour covers the whole line either way.
	muted := m.theme.Archived
	switch {
	case row.Orphan:
		muted = m.theme.Orphan
	case !s.Archived:
		muted = lipgloss.Style{}
	}
	shaded := func(own lipgloss.Style) lipgloss.Style {
		if muted.String() != "" {
			return muted
		}
		return own
	}

	// A sub-agent ran inside the conversation drawn above it, in the same
	// directory and within the same minute or two. Repeating all three columns
	// on its row says nothing and buries the tree; the space goes to the title.
	if row.Nested {
		line.Space(dayWidth + 1 + clockWidth + 2 + projectWidth + 1 + sizeWidth + 1)
	} else {
		// The project leads: it tells the reader which workspace a row is in
		// before the time it ran, which scans faster than the other way around.
		line.Cell(label, dayWidth, shaded(m.theme.Day)).Space(1)
		line.Cell(projectOf(s), projectWidth, shaded(m.theme.Project)).Space(2)
		line.Cell(s.RecencyAt().Format("15:04"), clockWidth, shaded(m.theme.Clock)).Space(1)
		line.Cell(formatBytes(s.Size), sizeWidth, shaded(m.theme.Project)).Space(1)
	}

	for _, guide := range row.Guides {
		line.Add(theme.Guide(guide), shaded(m.theme.Guide))
	}
	// One client dominates each agent, so naming it on every row is noise;
	// only the exceptions get a badge.
	if s.Client != "" && s.Client != m.meta.DefaultClient {
		line.Add(s.Client+" · ", shaded(m.theme.Client))
	}

	line.Add(titleOf(m.print, s), shaded(m.theme.Title))

	line.Highlight(m.search.query, caseSensitive(m.search.query), m.theme.Highlight)
	return line.Render(width)
}

// rowFill is the background a row is drawn on. A selection outranks the
// cursor, because the rows an action is about to touch matter more than the
// one the cursor happens to rest on.
//
// The cursor dims while the conversation pane has the keys, which is half of
// how the interface says where they are going; the other half is the rule
// between the two panes.
//
// Every other row is plain screen. Rows already carry a tree, a date column
// and a title; banding them as well gives the eye a fourth pattern to sort
// through for no more information.
func (m *Model) rowFill(cursor, picked bool) lipgloss.Style {
	switch {
	case cursor && picked:
		return m.theme.RowPickedCursor
	case picked:
		return m.theme.RowPicked
	case cursor && m.focus == focusDetail:
		return m.theme.RowCursorIdle
	case cursor:
		return m.theme.RowCursor
	}
	return m.theme.Screen
}

// projectOf is the project a session belongs to, which is the last segment of
// its working directory or an inferred project's name. An inferred project
// outranks the recorded directory: it points at the project the session
// worked on rather than the home it was launched from. The inferred name
// already names the project, so it is used as-is.
func projectOf(s session.Session) string {
	if s.Project != "" {
		return s.Project
	}
	if s.Cwd == "" {
		return ""
	}
	return filepath.Base(s.Cwd)
}

// effectiveProjectKey is the working directory a session belongs to, used to
// group and label it. An inferred project wins so home-launched sessions sort
// under the project they actually touched; anything else falls back to the
// recorded directory.
func effectiveProjectKey(s session.Session) string {
	if s.Project != "" {
		return s.Project
	}
	return s.Cwd
}

// formatBytes renders a session's on-disk size as a right-aligned string that
// fits in sizeWidth cells, so the unit suffixes line up in a column. Bigger
// sessions cost more context when resumed, so the size is worth a glance in the
// list: a 20 MB session will replay thousands of messages into the model, while
// a 12 KB one barely registers.
//
// Units are binary (1024-based) because that matches what users see in a file
// listing; B/KB/MB/GB read the same in either catalogue, so this needs no i18n.
// One decimal is shown only when the integer part is a single digit — that keeps
// "1.5MB" informative but never lets a column overflow (the largest form is
// "1023KB", still four cells). The result is then left-padded to sizeWidth.
func formatBytes(b int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
	)
	var body string
	switch {
	case b <= 0:
		// An empty or missing transcript has no weight to show.
		body = "—"
	case b < kb:
		body = strconv.FormatInt(b, 10) + "B"
	case b < mb:
		body = strconv.FormatInt(b/kb, 10) + "KB"
	case b < gb:
		body = scaled(float64(b)/mb, "MB")
	default:
		body = scaled(float64(b)/gb, "GB")
	}
	if gap := sizeWidth - text.Width(body); gap > 0 {
		body = strings.Repeat(" ", gap) + body
	}
	return body
}

// scaled formats a value with a unit suffix, using one decimal only when the
// integer part is a single digit (so 1.5MB but 20MB, never 1023.6MB).
func scaled(value float64, unit string) string {
	const oneDecimal = 10.0
	formatted := strconv.FormatFloat(value, 'f', 0, 64)
	if value < oneDecimal {
		formatted = strings.TrimSuffix(strconv.FormatFloat(value, 'f', 1, 64), ".0")
	}
	return formatted + unit
}
