package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// EventType distinguishes team vs individual events.
type EventType string

const (
	TeamEvent       EventType = "team"
	IndividualEvent EventType = "individual"
)

// Result represents a fully parsed NCTSA results PDF.
// Exactly one of TeamResult or IndividualResult will be non-nil.
type Result struct {
	EventName  string
	EventType  EventType
	Team       *TeamResult
	Individual *IndividualResult
}

// TeamResult holds parsed data from a team event PDF.
type TeamResult struct {
	Participants []TeamParticipant
}

// TeamParticipant represents a team entry parsed from a XXXXX-N ID.
type TeamParticipant struct {
	SchoolID   string
	TeamNumber string
	School     string
}

// IndividualResult holds parsed data from an individual event PDF.
type IndividualResult struct {
	Participants []IndividualParticipant
}

// IndividualParticipant represents an individual entry parsed from an 8-digit ID.
type IndividualParticipant struct {
	ID     string
	School string
}

var (
	teamIDPattern       = regexp.MustCompile(`^\d{5}-\d+$`)
	individualIDPattern = regexp.MustCompile(`^\d{7,8}$`)
	anyIDPattern        = regexp.MustCompile(`^(\d{5}-\d+|\d{7,8})$`)
	titleAbbrevPattern  = regexp.MustCompile(`\([A-Z]{2,}\)`)
	pageFooterPattern   = regexp.MustCompile(`(?i)page \d+ of \d+`)
)

// Parse reads an NCTSA results PDF and returns structured data.
// Returns an error if the file cannot be read or does not match the expected format.
func Parse(pdfPath string) (Result, error) {
	text, err := extractText(pdfPath)
	if err != nil {
		return Result{}, fmt.Errorf("extracting text: %w", err)
	}

	if issues := verify(text); len(issues) > 0 {
		return Result{}, fmt.Errorf("format verification failed: %s", strings.Join(issues, "; "))
	}

	eventType := classifyEvent(text)
	eventName := parseEventName(text)
	cleaned := cleanLines(text)

	switch eventType {
	case TeamEvent:
		participants, err := parseTeamParticipants(cleaned)
		if err != nil {
			return Result{}, fmt.Errorf("parsing team participants: %w", err)
		}
		return Result{
			EventName: eventName,
			EventType: TeamEvent,
			Team:      &TeamResult{Participants: participants},
		}, nil
	case IndividualEvent:
		participants, err := parseIndividualParticipants(cleaned)
		if err != nil {
			return Result{}, fmt.Errorf("parsing individual participants: %w", err)
		}
		return Result{
			EventName:  eventName,
			EventType:  IndividualEvent,
			Individual: &IndividualResult{Participants: participants},
		}, nil
	default:
		return Result{}, fmt.Errorf("unknown event type: %s", eventType)
	}
}

// Verify checks if a PDF file matches the NCTSA results format.
// Returns nil if valid, or a list of human-readable issues.
func Verify(pdfPath string) []string {
	text, err := extractText(pdfPath)
	if err != nil {
		return []string{fmt.Sprintf("failed to extract text: %v", err)}
	}
	return verify(text)
}

func extractText(pdfPath string) (string, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("opening PDF: %w", err)
	}
	defer f.Close()

	var allText strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("extracting text from page %d: %w", i, err)
		}
		allText.WriteString(text)
	}

	return allText.String(), nil
}

// verify checks that extracted PDF text matches the NCTSA results format.
func verify(text string) []string {
	var issues []string

	lines := cleanLines(text)
	if len(lines) == 0 {
		return []string{"PDF contains no extractable text"}
	}

	joined := strings.Join(lines, " ")
	upper := strings.ToUpper(joined)

	// 1. Title: event name with parenthesized abbreviation e.g. "(HS)"
	if !titleAbbrevPattern.MatchString(upper) {
		issues = append(issues, "missing title with parenthesized event abbreviation, e.g. (HS)")
	}

	// 2. Subtitle: result type keyword
	//    Small caps rendering fragments words across lines (e.g. "F\nINALISTS"),
	//    so strip all whitespace before checking.
	compacted := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, upper)
	resultKeywords := []string{"FINALISTS", "SEMI-FINALISTS", "WINNERS", "RESULTS", "QUALIFIERS"}
	hasResultType := false
	for _, kw := range resultKeywords {
		if strings.Contains(compacted, strings.ReplaceAll(kw, "-", "")) {
			hasResultType = true
			break
		}
		if strings.Contains(compacted, kw) {
			hasResultType = true
			break
		}
	}
	if !hasResultType {
		issues = append(issues, "missing result type subtitle (e.g. SEMI-FINALISTS, FINALISTS, WINNERS)")
	}

	// 3. Table headers: must contain all three column names
	headers := []string{"Participant ID", "School", "Division"}
	for _, h := range headers {
		found := false
		for _, line := range lines {
			if line == h {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("missing table header %q", h))
		}
	}

	// 4. Data rows: at least one participant ID (team: XXXXX-N, individual: XXXXXXXX)
	idCount := 0
	for _, line := range lines {
		if anyIDPattern.MatchString(line) {
			idCount++
		}
	}
	if idCount == 0 {
		issues = append(issues, "no participant ID rows found (expected format: XXXXX-N or XXXXXXXX)")
	}

	// 5. Each participant ID should be followed by a non-ID school name line
	for i, line := range lines {
		if anyIDPattern.MatchString(line) {
			if i+1 >= len(lines) {
				issues = append(issues, fmt.Sprintf("participant ID %s has no school name following it", line))
			} else if anyIDPattern.MatchString(lines[i+1]) {
				issues = append(issues, fmt.Sprintf("participant ID %s is followed by another ID instead of a school name", line))
			}
		}
	}

	// 6. Page footer
	if !pageFooterPattern.MatchString(joined) {
		issues = append(issues, "missing page footer (e.g. Page 1 of 1)")
	}

	return issues
}

// classifyEvent returns TeamEvent if the PDF uses XXXXX-N IDs, IndividualEvent for XXXXXXXX IDs.
func classifyEvent(text string) EventType {
	for _, line := range cleanLines(text) {
		if teamIDPattern.MatchString(line) {
			return TeamEvent
		}
		if individualIDPattern.MatchString(line) {
			return IndividualEvent
		}
	}
	return TeamEvent
}

// parseEventName reconstructs the event name from the fragmented small-caps title.
// It concatenates all raw lines up to and including the "(XX)" abbreviation line,
// then strips the parenthesized abbreviation itself.
func parseEventName(text string) string {
	var titleParts []string
	for _, line := range strings.Split(text, "\n") {
		// Whitespace-only lines act as word separators in the small-caps rendering.
		if strings.TrimSpace(line) == "" {
			if len(titleParts) > 0 {
				titleParts = append(titleParts, " ")
			}
			continue
		}
		// Preserve leading spaces — they indicate word boundaries.
		// A line like " T" starts a new word; "EAM" continues the previous.
		titleParts = append(titleParts, line)
		if titleAbbrevPattern.MatchString(strings.ToUpper(line)) {
			break
		}
	}

	raw := strings.Join(titleParts, "")

	// Remove the parenthesized abbreviation, e.g. "(HS)"
	name := titleAbbrevPattern.ReplaceAllString(raw, "")
	name = strings.TrimSpace(name)

	// Normalize internal whitespace
	name = strings.Join(strings.Fields(name), " ")

	// Title case: "HS CHAPTER TEAM" -> "HS Chapter Team"
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) <= 2 {
			// Keep short tokens uppercase (HS, MS, OF, etc. are fine as-is)
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}

	return strings.Join(words, " ")
}

func parseTeamParticipants(lines []string) ([]TeamParticipant, error) {
	var participants []TeamParticipant

	for i := 0; i < len(lines); i++ {
		if teamIDPattern.MatchString(lines[i]) {
			parts := strings.SplitN(lines[i], "-", 2)
			p := TeamParticipant{
				SchoolID:   parts[0],
				TeamNumber: parts[1],
			}
			if i+1 < len(lines) && !anyIDPattern.MatchString(lines[i+1]) {
				p.School = lines[i+1]
				i++
			}
			participants = append(participants, p)
		}
	}

	if len(participants) == 0 {
		return nil, fmt.Errorf("no team participant rows found")
	}

	return participants, nil
}

func parseIndividualParticipants(lines []string) ([]IndividualParticipant, error) {
	var participants []IndividualParticipant

	for i := 0; i < len(lines); i++ {
		if individualIDPattern.MatchString(lines[i]) {
			p := IndividualParticipant{ID: lines[i]}
			if i+1 < len(lines) && !anyIDPattern.MatchString(lines[i+1]) {
				p.School = lines[i+1]
				i++
			}
			participants = append(participants, p)
		}
	}

	if len(participants) == 0 {
		return nil, fmt.Errorf("no individual participant rows found")
	}

	return participants, nil
}

func cleanLines(text string) []string {
	var cleaned []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return cleaned
}
