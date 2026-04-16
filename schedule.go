package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// ScheduleResult holds parsed data from a schedule PDF.
type ScheduleResult struct {
	Title   string
	Date    string
	Level   string
	Entries []ScheduleEntry
}

// ScheduleEntry represents a single participant's schedule entry.
type ScheduleEntry struct {
	Name         string
	School       string
	IndividualID string
	TeamID       string
	Section      string
	HoldTime     string
	PrepTime     string
	PresentTime  string
	EndTime      string
}

var (
	dateLvlPattern    = regexp.MustCompile(`^(\d{1,2}/\d{1,2}/\d{4})\s+Lvl:\s*(\d+)`)
	partialTeamID     = regexp.MustCompile(`^\d{5}-$`)
	fullTeamIDPattern = regexp.MustCompile(`^\d{5}-\d+$`)
	sectionPattern    = regexp.MustCompile(`^\d{1,2}$`)
	timePattern       = regexp.MustCompile(`^\d{1,2}:\d{2}$`)
	ampmPattern       = regexp.MustCompile(`(?i)^[AP]M$`)
	timeFullPattern   = regexp.MustCompile(`^\d{1,2}:\d{2}\s*[AP]M$`)
)

func parseScheduleFile(pdfPath string, ft FileType) (Result, error) {
	text, err := extractText(pdfPath)
	if err != nil {
		return Result{}, fmt.Errorf("extracting text: %w", err)
	}

	switch ft {
	case FileScheduleClean:
		sched, err := parseScheduleClean(text)
		if err != nil {
			return Result{}, fmt.Errorf("parsing clean schedule: %w", err)
		}
		return Result{
			EventName: sched.Title,
			FileType:  FileScheduleClean,
			Schedule:  sched,
		}, nil
	case FileScheduleRaw:
		sched, err := parseScheduleRaw(text)
		if err != nil {
			return Result{}, fmt.Errorf("parsing raw schedule: %w", err)
		}
		return Result{
			EventName: sched.Title,
			FileType:  FileScheduleRaw,
			Schedule:  sched,
		}, nil
	default:
		return Result{}, fmt.Errorf("unexpected file type for schedule: %s", ft)
	}
}

func parseScheduleClean(text string) (*ScheduleResult, error) {
	lines := cleanLines(text)
	if len(lines) < 3 {
		return nil, fmt.Errorf("too few lines for schedule")
	}

	title := lines[0]

	matches := dateLvlPattern.FindStringSubmatch(lines[1])
	if matches == nil {
		return nil, fmt.Errorf("second line does not match date/level pattern: %q", lines[1])
	}
	date := matches[1]
	level := matches[2]

	// Skip garbled header lines. Find the first line that looks like a participant name:
	// not an ID, not garbled, not a time, not a section number, contains letters.
	dataStart := 2
	for dataStart < len(lines) {
		line := lines[dataStart]
		if isGarbledLine(line) || line == "BREAK" {
			dataStart++
			continue
		}
		// Once we find a line with mostly alphabetic chars that isn't an ID or time, we're in data
		if !anyIDPattern.MatchString(line) && !individualIDPattern.MatchString(line) &&
			!timePattern.MatchString(line) && !ampmPattern.MatchString(line) &&
			!sectionPattern.MatchString(line) && !timeFullPattern.MatchString(line) &&
			hasLetters(line) {
			break
		}
		dataStart++
	}

	// Preprocess: join split team IDs (e.g., "21473-" + "1" -> "21473-1")
	processed := joinSplitTeamIDs(lines[dataStart:])

	var entries []ScheduleEntry
	i := 0
	for i < len(processed) {
		line := processed[i]

		// Skip non-data lines
		if isGarbledLine(line) || line == "BREAK" || pageFooterPattern.MatchString(line) {
			i++
			continue
		}

		// Skip timestamp/browser header lines from PDF export
		if strings.Contains(line, "Administration - All Schedule") {
			i++
			continue
		}

		// Skip page number lines like "1/4", "2/4"
		if regexp.MustCompile(`^\d+\s*/\s*\d+$`).MatchString(line) {
			i++
			continue
		}

		// Skip lines that look like URL fragments
		if strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "http://") {
			i++
			continue
		}

		// Skip date/time header lines from PDF rendering
		if regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{2,4},\s*\d`).MatchString(line) {
			i++
			continue
		}

		// Look for a participant entry: alphabetic lines followed by an individual ID.
		// Use ID-anchoring: collect all text lines up to the next individual ID,
		// then split into name (first part) and school (last part before ID).
		if hasLetters(line) && !anyIDPattern.MatchString(line) && !individualIDPattern.MatchString(line) &&
			!timePattern.MatchString(line) && !ampmPattern.MatchString(line) &&
			!timeFullPattern.MatchString(line) && !sectionPattern.MatchString(line) {

			// Collect all text lines until we hit an individual ID
			var textLines []string
			for i < len(processed) && hasLetters(processed[i]) &&
				!individualIDPattern.MatchString(processed[i]) &&
				!anyIDPattern.MatchString(processed[i]) {
				textLines = append(textLines, processed[i])
				i++
			}

			// Split textLines into name and school using the school boundary.
			// The school occupies the trailing lines; the name occupies the leading lines.
			name, school := splitNameAndSchool(textLines)

			entry := ScheduleEntry{Name: name, School: school}

			// Individual ID (8-digit)
			if i < len(processed) && individualIDPattern.MatchString(processed[i]) {
				entry.IndividualID = processed[i]
				i++
			}

			// Optional Team ID or Section
			if i < len(processed) {
				if fullTeamIDPattern.MatchString(processed[i]) {
					entry.TeamID = processed[i]
					i++
					// Section follows team ID
					if i < len(processed) && sectionPattern.MatchString(processed[i]) {
						entry.Section = processed[i]
						i++
					}
				} else if sectionPattern.MatchString(processed[i]) {
					// No team ID, direct to section
					entry.Section = processed[i]
					i++
				}
			}

			// 4 time values: Hold, Prep, Present, End
			times := readTimes(processed, &i)
			if len(times) >= 4 {
				entry.HoldTime = times[0]
				entry.PrepTime = times[1]
				entry.PresentTime = times[2]
				entry.EndTime = times[3]
			}

			entries = append(entries, entry)
			continue
		}

		i++
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no schedule entries found")
	}

	return &ScheduleResult{
		Title:   title,
		Date:    date,
		Level:   level,
		Entries: entries,
	}, nil
}

func parseScheduleRaw(text string) (*ScheduleResult, error) {
	lines := cleanLines(text)
	if len(lines) < 5 {
		return nil, fmt.Errorf("too few lines for raw schedule")
	}

	sched := &ScheduleResult{}

	// The raw export renders the event name on line 1 using a shifted font encoding
	// (each byte = target - 29). Decode it so callers get e.g. "HS Digital Video Production"
	// rather than the "Schedule Name" subtitle like "Finalist Interviews". Fall back to
	// Schedule Name if the decode doesn't look sensible.
	if decoded := decodeShiftedTitle(lines[0]); decoded != "" {
		sched.Title = decoded
	}

	// Extract metadata from "Schedule Name:", "Schedule Date:" lines. Only overwrite the
	// title from Schedule Name when we couldn't recover a proper event name from line 1.
	for _, line := range lines {
		if after, ok := strings.CutPrefix(line, "Schedule Name: "); ok {
			if sched.Title == "" {
				sched.Title = after
			}
		}
		if after, ok := strings.CutPrefix(line, "Schedule Date: "); ok {
			sched.Date = after
		}
	}

	// Preprocess: join split team IDs
	processed := joinSplitTeamIDs(lines)

	// Parse team groups. Each group has:
	// - Individual entries: Last, First, ID (repeated per team member)
	// - School name (multi-line)
	// - Team ID
	// - Times
	// - "Edit"

	i := 0
	// Skip metadata header
	for i < len(processed) {
		if processed[i] == "Edit" || (sectionPattern.MatchString(processed[i]) && i+1 < len(processed) && isGarbledLine(processed[i+1])) {
			break
		}
		i++
	}

	// Filter out noise lines (garbled, page markers, URLs, section numbers, etc.)
	// to get clean data lines containing only: names, IDs, school words, team IDs, times.
	var dataLines []string
	for idx, line := range processed {
		if isSkippableLine(line) {
			continue
		}
		// Skip standalone section numbers (1-2 digits) that appear between groups.
		// A section number is distinguished from a school word by context:
		// it appears right after an "Edit" or garbled line, or at the very start.
		if sectionPattern.MatchString(line) && !timePattern.MatchString(line) {
			// Check if this could be a time component like part of "3:00"
			// If it's just a bare number between non-numeric lines, it's a section number.
			isAfterNoise := idx == 0
			if idx > 0 {
				prev := processed[idx-1]
				isAfterNoise = isSkippableLine(prev) || prev == "Edit"
			}
			isBeforeNoise := idx+1 >= len(processed)
			if idx+1 < len(processed) {
				next := processed[idx+1]
				isBeforeNoise = isSkippableLine(next)
			}
			if isAfterNoise || isBeforeNoise {
				continue
			}
		}
		dataLines = append(dataLines, line)
	}

	// ID-anchored parsing: find all individual IDs first, then work outward.
	// In Format B, each team group is:
	//   [name lines]... ID [name lines]... ID ... SchoolWords... TeamID Time*4
	// The line immediately before each ID is the first name; everything between
	// the previous ID and this first name is the (possibly multi-word) last name.
	type member struct {
		firstName string
		lastName  string
		id        string
	}

	// Collect groups by scanning for team IDs (group terminators).
	// Split member runs at ID prefix boundaries to handle page-break orphans.
	type teamGroup struct {
		members []member
		school  string
		teamID  string
		times   []string
	}

	var groups []teamGroup
	di := 0
	for di < len(dataLines) {
		// Collect all individual IDs in this run (before next school+teamID block)
		var idPositions []int
		scanStart := di
		j := di
		for j < len(dataLines) {
			if individualIDPattern.MatchString(dataLines[j]) {
				idPositions = append(idPositions, j)
			} else if fullTeamIDPattern.MatchString(dataLines[j]) {
				break
			} else if timePattern.MatchString(dataLines[j]) || timeFullPattern.MatchString(dataLines[j]) ||
				ampmPattern.MatchString(dataLines[j]) {
				break
			}
			j++
		}

		if len(idPositions) == 0 {
			di++
			continue
		}

		// Build all members from ID positions.
		// Track ID prefix changes to detect boundaries between groups
		// where leftover school-name text may appear.
		var allMembers []member
		prevBoundary := scanStart
		prevPrefix := ""
		for _, p := range idPositions {
			if p < 1 {
				continue
			}
			currentPrefix := ""
			if len(dataLines[p]) >= 5 {
				currentPrefix = dataLines[p][:5]
			}

			firstName := dataLines[p-1]

			// Collect last name parts between prevBoundary and p-1
			var lastParts []string
			for k := prevBoundary; k < p-1; k++ {
				if !individualIDPattern.MatchString(dataLines[k]) {
					lastParts = append(lastParts, dataLines[k])
				}
			}

			// At prefix boundaries (different team), leftover school-name text
			// from the previous team may precede the actual last name.
			// Keep only the last line(s) as the actual last name.
			if prevPrefix != "" && currentPrefix != prevPrefix && len(lastParts) > 1 {
				lastParts = lastParts[len(lastParts)-1:]
			}

			// Join last name parts, handling hyphenated names (e.g., "Scotten-" + "White" → "Scotten-White")
			lastName := joinHyphenatedParts(lastParts)
			allMembers = append(allMembers, member{firstName: firstName, lastName: lastName, id: dataLines[p]})
			prevBoundary = p + 1
			prevPrefix = currentPrefix
		}

		di = prevBoundary

		// Split members into sub-groups at ID prefix boundaries.
		// All members with the same 5-digit prefix belong to the same team.
		var subGroups [][]member
		var currentSub []member
		var currentPrefix string
		for _, m := range allMembers {
			prefix := ""
			if len(m.id) >= 5 {
				prefix = m.id[:5]
			}
			if prefix != currentPrefix && len(currentSub) > 0 {
				subGroups = append(subGroups, currentSub)
				currentSub = nil
			}
			currentPrefix = prefix
			currentSub = append(currentSub, m)
		}
		if len(currentSub) > 0 {
			subGroups = append(subGroups, currentSub)
		}

		// Consume school name lines, team ID, and times (these belong to the LAST sub-group)
		var schoolParts []string
		for di < len(dataLines) {
			l := dataLines[di]
			if fullTeamIDPattern.MatchString(l) || timePattern.MatchString(l) || timeFullPattern.MatchString(l) ||
				ampmPattern.MatchString(l) || sectionPattern.MatchString(l) || individualIDPattern.MatchString(l) {
				break
			}
			schoolParts = append(schoolParts, l)
			di++
		}
		school := strings.Join(schoolParts, " ")

		teamID := ""
		if di < len(dataLines) && fullTeamIDPattern.MatchString(dataLines[di]) {
			teamID = dataLines[di]
			di++
		}

		times := readTimes(dataLines, &di)

		// Process sub-groups: the last sub-group gets the school/team/times we just read.
		// Earlier sub-groups are orphans from page breaks — carry forward from previous group.
		hasOrphans := false
		for si, sub := range subGroups {
			if si < len(subGroups)-1 {
				hasOrphans = true
				if len(groups) > 0 {
					prev := groups[len(groups)-1]
					groups = append(groups, teamGroup{members: sub, school: prev.school, teamID: prev.teamID, times: prev.times})
				} else {
					groups = append(groups, teamGroup{members: sub})
				}
			} else {
				groups = append(groups, teamGroup{members: sub, school: school, teamID: teamID, times: times})
			}
		}

		// After orphan groups, leftover school-name fragments from the page break
		// may appear in the data lines (e.g., "Science", "Academy" from a split school name).
		// Skip these before continuing to the next group.
		if hasOrphans {
			for di < len(dataLines) {
				l := dataLines[di]
				// Stop if we hit an ID, time, or team ID — real data ahead
				if individualIDPattern.MatchString(l) || fullTeamIDPattern.MatchString(l) ||
					timePattern.MatchString(l) || timeFullPattern.MatchString(l) || ampmPattern.MatchString(l) {
					break
				}
				// If this line looks like school text and is NOT followed by a first-name + ID triple,
				// it's leftover school text. Check by looking 2 ahead for an ID.
				if hasLetters(l) && di+2 < len(dataLines) && !individualIDPattern.MatchString(dataLines[di+2]) {
					di++
					continue
				}
				break
			}
		}
	}

	// Flatten groups into entries
	for _, g := range groups {
		for _, m := range g.members {
			entry := ScheduleEntry{
				Name:         m.firstName + " " + m.lastName,
				School:       g.school,
				IndividualID: m.id,
				TeamID:       g.teamID,
			}
			if len(g.times) >= 4 {
				entry.HoldTime = g.times[0]
				entry.PrepTime = g.times[1]
				entry.PresentTime = g.times[2]
				entry.EndTime = g.times[3]
			}
			sched.Entries = append(sched.Entries, entry)
		}
	}

	if len(sched.Entries) == 0 {
		return nil, fmt.Errorf("no schedule entries found")
	}

	return sched, nil
}

// readTimes reads up to 4 time values from the line list.
// Times can be "H:MM AM" on one line or "H:MM" + "AM" on separate lines.
func readTimes(lines []string, i *int) []string {
	var times []string
	for len(times) < 4 && *i < len(lines) {
		line := lines[*i]

		// Full time on one line: "9:05 AM"
		if timeFullPattern.MatchString(line) {
			times = append(times, line)
			*i++
			continue
		}

		// Split time: "9:00" + "AM"
		if timePattern.MatchString(line) && *i+1 < len(lines) && ampmPattern.MatchString(lines[*i+1]) {
			times = append(times, line+" "+strings.ToUpper(lines[*i+1]))
			*i += 2
			continue
		}

		// Not a time — stop
		break
	}
	return times
}

// joinSplitTeamIDs preprocesses lines to rejoin team IDs split across lines
// (e.g., "21473-" + "1" -> "21473-1").
func joinSplitTeamIDs(lines []string) []string {
	var result []string
	for i := 0; i < len(lines); i++ {
		if partialTeamID.MatchString(lines[i]) && i+1 < len(lines) {
			result = append(result, lines[i]+lines[i+1])
			i++
		} else {
			result = append(result, lines[i])
		}
	}
	return result
}

var (
	pageNumPattern     = regexp.MustCompile(`^\d+\s*/\s*\d+$`)
	dateTimeHdrPattern = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{2,4},\s*\d`)
)

// isSkippableLine returns true for lines that should be filtered out of schedule data.
func isSkippableLine(line string) bool {
	if isGarbledLine(line) {
		return true
	}
	switch line {
	case "Edit", "Swap", "Insert Break", "BREAK":
		return true
	}
	if pageFooterPattern.MatchString(line) ||
		strings.HasPrefix(line, "https://") ||
		strings.HasPrefix(line, "http://") ||
		strings.Contains(line, "Administration - All Schedule") ||
		strings.HasPrefix(line, "Displays all") ||
		strings.HasPrefix(line, "Schedule Name:") ||
		strings.HasPrefix(line, "Schedule Date:") ||
		strings.HasPrefix(line, "Schedule Time Range:") {
		return true
	}
	if pageNumPattern.MatchString(line) || dateTimeHdrPattern.MatchString(line) {
		return true
	}
	// Single-character non-alphanumeric lines (e.g., "/" from split page numbers "1/3")
	if len(line) == 1 && !hasLetters(line) && (line[0] < '0' || line[0] > '9') {
		return true
	}
	return false
}

// isGarbledLine detects font-encoded garbled text from PDF exports.
func isGarbledLine(line string) bool {
	if len(line) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Lines containing control characters (bytes < 0x20, except tab/newline/CR)
	// are garbled PDF encoding artifacts.
	for _, b := range []byte(trimmed) {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			return true
		}
	}

	// Known garbled tokens from the registermychapter.com PDF exports.
	// Check if the line contains any of these as substrings.
	garbledTokens := []string{
		"3DUWLFLSDQW", "7HDP", "6HFWLRQ", "+ROG", "7LPH", "3UHS",
		"3UHVHQW", "(QG", "6FKRRO", "6HF", "3DUWLFLSDQWV",
		"/DVW", ")LUVW", ",QGLYLGXDO", "0667", ",'",
	}
	for _, g := range garbledTokens {
		if strings.Contains(trimmed, g) {
			return true
		}
	}

	return false
}

// splitNameAndSchool separates a buffer of text lines into a participant name
// and school name. The school is the trailing portion; the name is the leading portion.
// Uses heuristics: school lines typically contain organizational words like
// "School", "Academy", "High", "Middle", "College", "Charter".
func splitNameAndSchool(lines []string) (name, school string) {
	if len(lines) == 0 {
		return "", ""
	}
	if len(lines) == 1 {
		return lines[0], ""
	}

	// Find where the school starts by scanning backwards from the end.
	// The school is the longest trailing run that contains school-type indicators
	// or is a continuation word of a school name.
	schoolStart := len(lines)
	for j := len(lines) - 1; j >= 1; j-- {
		if isSchoolLine(lines[j]) || isSchoolContinuation(lines[j]) {
			schoolStart = j
		} else {
			break
		}
	}

	// If no school indicator found, assume last line is school, rest is name
	if schoolStart == len(lines) {
		schoolStart = len(lines) - 1
	}

	// Ensure at least 1 line for the name
	if schoolStart < 1 {
		schoolStart = 1
	}

	// Handle hyphenated names: join name parts, preserving hyphens
	nameParts := lines[:schoolStart]
	nameStr := nameParts[0]
	for k := 1; k < len(nameParts); k++ {
		if strings.HasSuffix(nameStr, "-") {
			nameStr += nameParts[k]
		} else {
			nameStr += " " + nameParts[k]
		}
	}

	return nameStr, strings.Join(lines[schoolStart:], " ")
}

// isSchoolLine returns true if the line contains school/organization type words.
func isSchoolLine(s string) bool {
	lower := strings.ToLower(s)
	indicators := []string{
		"school", "academy", "academic", "high", "middle", "college",
		"charter", "magnet", "institute", "technology", "technical",
		"preparatory", "prep", "arts", "stem", "science", "math",
		"carolina", "morganton", "early", "information",
	}
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}
	return false
}

// isSchoolContinuation returns true if the line starts with a common
// continuation word in school names (e.g., "And Mathematics - Morganton").
func isSchoolContinuation(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	// Check if the first word is a continuation word
	firstWord := strings.Fields(lower)
	if len(firstWord) == 0 {
		return false
	}
	continuations := []string{"of", "and", "the", "for", "&", "-"}
	for _, c := range continuations {
		if firstWord[0] == c {
			return true
		}
	}
	return false
}

// joinHyphenatedParts joins string parts, preserving hyphens without adding spaces.
// e.g., ["Scotten-", "White"] → "Scotten-White", ["Coronilla", "Penaloza"] → "Coronilla Penaloza"
func joinHyphenatedParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if strings.HasSuffix(result, "-") {
			result += parts[i]
		} else {
			result += " " + parts[i]
		}
	}
	return result
}

// decodeShiftedTitle recovers the event title from the raw schedule export's first line.
// registermychapter.com renders the event name with a font whose glyph codes sit 29 below
// plain ASCII, so bytes land as e.g. 0x2B/0x36 ("+6") for "HS". We add 29 to each byte and
// strip the trailing " - Level: N" suffix. Returns "" if the decode doesn't look like an
// event name, letting callers fall back to the "Schedule Name" line.
func decodeShiftedTitle(raw string) string {
	if raw == "" {
		return ""
	}
	b := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c < 0x20 || (c >= 0x21 && c <= 0x7A) {
			c += 29
		}
		b = append(b, c)
	}
	decoded := strings.TrimSpace(string(b))
	// Strip " - Level: N" and similar suffix
	if idx := strings.Index(decoded, " - Level"); idx > 0 {
		decoded = strings.TrimSpace(decoded[:idx])
	}
	// Sanity check: should look like "HS Something" or "MS Something"
	upper := strings.ToUpper(decoded)
	if !strings.HasPrefix(upper, "HS ") && !strings.HasPrefix(upper, "MS ") {
		return ""
	}
	return decoded
}

// hasLetters returns true if the string contains at least one letter.
func hasLetters(s string) bool {
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			return true
		}
	}
	return false
}
