package parser

import (
	"strings"
	"testing"
)

func TestVerify_ValidTeamText(t *testing.T) {
	text := `HS C
HAPTER
 T
EAM
 (HS)
S
EMI
-F
INALISTS
 -
IN
 R
ANDOM

ORDER
Participant ID
School
Division
21333-1
Panther Creek High School
26510-1
Ronald Wilson Reagan High School
Page 1 of 1
`
	errs := verify(text)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid team text, got: %v", errs)
	}
}

func TestVerify_ValidIndividualText(t *testing.T) {
	text := `HS P
HOTOGRAPHIC
 T
ECHNOLOGY
 (HS)
S
EMI
-F
INALISTS
 -
IN
 R
ANDOM

ORDER
Participant ID
School
Division
26432007
Woodlawn School
24694086
Apex Friendship High School
Page 1 of 1
`
	errs := verify(text)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid individual text, got: %v", errs)
	}
}

func TestVerify_MissingTitle(t *testing.T) {
	text := `Some Random Title
SEMI-FINALISTS
Participant ID
School
Division
21333-1
Panther Creek High School
Page 1 of 1
`
	errs := verify(text)
	assertContainsIssue(t, errs, "missing title with parenthesized event abbreviation")
}

func TestVerify_MissingResultType(t *testing.T) {
	text := `Event Name (HS)
Participant ID
School
Division
21333-1
Panther Creek High School
Page 1 of 1
`
	errs := verify(text)
	assertContainsIssue(t, errs, "missing result type subtitle")
}

func TestVerify_MissingHeaders(t *testing.T) {
	text := `Event (HS)
FINALISTS
21333-1
Panther Creek High School
Page 1 of 1
`
	errs := verify(text)
	assertContainsIssue(t, errs, `missing table header "Participant ID"`)
	assertContainsIssue(t, errs, `missing table header "School"`)
	assertContainsIssue(t, errs, `missing table header "Division"`)
}

func TestVerify_NoDataRows(t *testing.T) {
	text := `Event (HS)
FINALISTS
Participant ID
School
Division
Page 1 of 1
`
	errs := verify(text)
	assertContainsIssue(t, errs, "no participant ID rows found")
}

func TestVerify_MissingSchoolAfterID(t *testing.T) {
	text := `Event (HS)
FINALISTS
Participant ID
School
Division
21333-1
21334-1
Some School
Page 1 of 1
`
	errs := verify(text)
	assertContainsIssue(t, errs, "participant ID 21333-1 is followed by another ID")
}

func TestVerify_MissingPageFooter(t *testing.T) {
	text := `Event (HS)
FINALISTS
Participant ID
School
Division
21333-1
Panther Creek High School
`
	errs := verify(text)
	assertContainsIssue(t, errs, "missing page footer")
}

func TestVerify_EmptyText(t *testing.T) {
	errs := verify("")
	assertContainsIssue(t, errs, "no extractable text")
}

func TestParseEventName(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name: "team event - chapter",
			text: "HS C\nHAPTER\n T\nEAM\n (HS)\nS\nEMI\n",
			expected: "HS Chapter Team",
		},
		{
			name: "individual event - photo",
			text: "HS P\nHOTOGRAPHIC\n T\nECHNOLOGY\n (HS)\nS\nEMI\n",
			expected: "HS Photographic Technology",
		},
		{
			name: "multi-word - future tech teacher",
			text: "HS F\nUTURE\n T\nECHNOLOGY\n \nAND\n E\nNGINEERING\n T\nEACHER\n (HS)\nS\n",
			expected: "HS Future Technology And Engineering Teacher",
		},
		{
			name: "middle school event",
			text: "MS C\nAREER\n P\nREP\n (MS)\nS\nEMI\n",
			expected: "MS Career Prep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEventName(tt.text)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestClassifyEvent_Team(t *testing.T) {
	text := `Participant ID
School
Division
21333-1
Panther Creek High School
`
	if got := classifyEvent(text); got != TeamEvent {
		t.Errorf("expected TeamEvent, got %s", got)
	}
}

func TestClassifyEvent_Individual(t *testing.T) {
	text := `Participant ID
School
Division
26432007
Woodlawn School
`
	if got := classifyEvent(text); got != IndividualEvent {
		t.Errorf("expected IndividualEvent, got %s", got)
	}
}

func TestParseTeamParticipants(t *testing.T) {
	lines := cleanLines(`Participant ID
School
Division
21333-1
Panther Creek High School
26510-2
Ronald Wilson Reagan High School
Page 1 of 1
`)
	participants, err := parseTeamParticipants(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}

	p0 := participants[0]
	if p0.SchoolID != "21333" || p0.TeamNumber != "1" || p0.School != "Panther Creek High School" {
		t.Errorf("unexpected first participant: %+v", p0)
	}

	p1 := participants[1]
	if p1.SchoolID != "26510" || p1.TeamNumber != "2" || p1.School != "Ronald Wilson Reagan High School" {
		t.Errorf("unexpected second participant: %+v", p1)
	}
}

func TestParseIndividualParticipants(t *testing.T) {
	lines := cleanLines(`Participant ID
School
Division
26432007
Woodlawn School
24694086
Apex Friendship High School
Page 1 of 1
`)
	participants, err := parseIndividualParticipants(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}
	if participants[0].ID != "26432007" || participants[0].School != "Woodlawn School" {
		t.Errorf("unexpected first participant: %+v", participants[0])
	}
	if participants[1].ID != "24694086" || participants[1].School != "Apex Friendship High School" {
		t.Errorf("unexpected second participant: %+v", participants[1])
	}
}

func assertContainsIssue(t *testing.T, errs []string, substr string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got: %v", substr, errs)
}
