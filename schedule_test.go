package parser

import "testing"

func TestParseScheduleClean_WithTeamID(t *testing.T) {
	text := `HS Chapter Team - Finalist Presentations
4/10/2026 Lvl: 2
3DUWLFLSDQW
,'
7HDP,'
6HFWLRQ
+ROG
7LPH
3UHS
7LPH
3UHVHQW
7LPH
(QG
7LPH
Christian Burgess
Phillip O. Berry Academy Of Technology
21473022
21473-1
1
9:00
AM
9:00
AM
9:05 AM
9:25
AM
Alondra Delgado-Martinez
Phillip O. Berry Academy Of Technology
21473045
21473-1
1
9:20
AM
9:20
AM
9:25 AM
9:45
AM
`
	sched, err := parseScheduleClean(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sched.Title != "HS Chapter Team - Finalist Presentations" {
		t.Errorf("expected title %q, got %q", "HS Chapter Team - Finalist Presentations", sched.Title)
	}
	if sched.Date != "4/10/2026" {
		t.Errorf("expected date 4/10/2026, got %q", sched.Date)
	}
	if sched.Level != "2" {
		t.Errorf("expected level 2, got %q", sched.Level)
	}
	if len(sched.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(sched.Entries))
	}

	e := sched.Entries[0]
	if e.Name != "Christian Burgess" {
		t.Errorf("expected name %q, got %q", "Christian Burgess", e.Name)
	}
	if e.School != "Phillip O. Berry Academy Of Technology" {
		t.Errorf("expected school %q, got %q", "Phillip O. Berry Academy Of Technology", e.School)
	}
	if e.IndividualID != "21473022" {
		t.Errorf("expected ID 21473022, got %q", e.IndividualID)
	}
	if e.TeamID != "21473-1" {
		t.Errorf("expected TeamID 21473-1, got %q", e.TeamID)
	}
	if e.HoldTime != "9:00 AM" {
		t.Errorf("expected HoldTime 9:00 AM, got %q", e.HoldTime)
	}
	if e.PresentTime != "9:05 AM" {
		t.Errorf("expected PresentTime 9:05 AM, got %q", e.PresentTime)
	}
	if e.EndTime != "9:25 AM" {
		t.Errorf("expected EndTime 9:25 AM, got %q", e.EndTime)
	}
}

func TestParseScheduleClean_NoTeamID(t *testing.T) {
	text := `HS Prepared Presentation - Prepared Presentation Times
4/10/2026 Lvl: 2
3DUWLFLSDQW
,'
7HDP,'
6HFWLRQ
+ROG
7LPH
3UHS
7LPH
3UHVHQW
7LPH
(QG
7LPH
Pavit Singh
Ardrey Kell High School
22391057
1
9:00
AM
9:00
AM
9:00 AM
9:05
AM
`
	sched, err := parseScheduleClean(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sched.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(sched.Entries))
	}

	e := sched.Entries[0]
	if e.Name != "Pavit Singh" {
		t.Errorf("expected name %q, got %q", "Pavit Singh", e.Name)
	}
	if e.TeamID != "" {
		t.Errorf("expected empty TeamID, got %q", e.TeamID)
	}
	if e.IndividualID != "22391057" {
		t.Errorf("expected ID 22391057, got %q", e.IndividualID)
	}
	if e.Section != "1" {
		t.Errorf("expected section 1, got %q", e.Section)
	}
}

func TestParseScheduleClean_HyphenatedName(t *testing.T) {
	text := `HS Chapter Team - Finalist Presentations
4/10/2026 Lvl: 2
3DUWLFLSDQW
Alondra Delgado-
Martinez
Phillip O. Berry Academy Of Technology
21473045
21473-1
1
9:00
AM
9:00
AM
9:05 AM
9:25
AM
`
	sched, err := parseScheduleClean(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sched.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(sched.Entries))
	}
	e := sched.Entries[0]
	if e.Name != "Alondra Delgado-Martinez" {
		t.Errorf("expected name %q, got %q", "Alondra Delgado-Martinez", e.Name)
	}
	if e.School != "Phillip O. Berry Academy Of Technology" {
		t.Errorf("expected school %q, got %q", "Phillip O. Berry Academy Of Technology", e.School)
	}
}

func TestParseScheduleClean_SplitTeamID(t *testing.T) {
	text := `HS Forensics - SEMIFINAL ROUND
4/10/2026 Lvl: 2
3DUWLFLSDQW
Sri Bondalapati
Triangle Math And Science Academy
26319009
26319-
2
1
1:00
PM
1:00
PM
1:20 PM
1:40
PM
`
	sched, err := parseScheduleClean(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sched.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(sched.Entries))
	}

	e := sched.Entries[0]
	if e.TeamID != "26319-2" {
		t.Errorf("expected TeamID 26319-2, got %q", e.TeamID)
	}
}

func TestParseScheduleRaw_Basic(t *testing.T) {
	text := `0667(0$QLPDWLRQ/HYHO
,QGLYLGXDO6FKHGXOHV
Displays all individual and team schedules for this event.
Schedule Name: Finalist Interviews
Schedule Date: 4/9/2026
Schedule Time Range: 3:00 PM to 6:00 PM
Swap
Insert Break
6HF
3DUWLFLSDQWV
6FKRRO
7HDP
+ROG
7LPH
3UHS
7LPH
3UHVHQW
7LPH
(QG
7LPH
1
/DVW
)LUVW
,'
Pecorella
Brennon
12309055
Peterson
Tramel
12309056
Hanes Magnet School
12309-1
3:00
PM
3:00
PM
3:00
PM
3:10
PM
Edit
`
	sched, err := parseScheduleRaw(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sched.Title != "Finalist Interviews" {
		t.Errorf("expected title %q, got %q", "Finalist Interviews", sched.Title)
	}
	if sched.Date != "4/9/2026" {
		t.Errorf("expected date 4/9/2026, got %q", sched.Date)
	}
	if len(sched.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(sched.Entries))
	}

	e0 := sched.Entries[0]
	if e0.Name != "Brennon Pecorella" {
		t.Errorf("expected name %q, got %q", "Brennon Pecorella", e0.Name)
	}
	if e0.IndividualID != "12309055" {
		t.Errorf("expected ID 12309055, got %q", e0.IndividualID)
	}
	if e0.School != "Hanes Magnet School" {
		t.Errorf("expected school %q, got %q", "Hanes Magnet School", e0.School)
	}
	if e0.TeamID != "12309-1" {
		t.Errorf("expected TeamID 12309-1, got %q", e0.TeamID)
	}
	if e0.HoldTime != "3:00 PM" {
		t.Errorf("expected HoldTime 3:00 PM, got %q", e0.HoldTime)
	}

	e1 := sched.Entries[1]
	if e1.Name != "Tramel Peterson" {
		t.Errorf("expected name %q, got %q", "Tramel Peterson", e1.Name)
	}
	if e1.IndividualID != "12309056" {
		t.Errorf("expected ID 12309056, got %q", e1.IndividualID)
	}
}

func TestParseScheduleRaw_MultiLineSchool(t *testing.T) {
	text := `0667(0$QLPDWLRQ/HYHO
,QGLYLGXDO6FKHGXOHV
Displays all individual and team schedules for this event.
Schedule Name: Finalist
Schedule Date: 4/9/2026
Schedule Time Range: 3:00 PM to 6:00 PM
Swap
Insert Break
6HF
3DUWLFLSDQWV
1
/DVW
)LUVW
,'
Rujubali
Nameer
16897013
Gaston
Day
School
MS
16897-1
3:20
PM
3:20
PM
3:20
PM
3:30
PM
Edit
`
	sched, err := parseScheduleRaw(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sched.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(sched.Entries))
	}

	e := sched.Entries[0]
	if e.School != "Gaston Day School MS" {
		t.Errorf("expected school %q, got %q", "Gaston Day School MS", e.School)
	}
}

func TestVerifyWithScores_Valid(t *testing.T) {
	text := `MS T
ECH
 B
OWL
 (MS)
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
Score
16897-2
Gaston Day School MS
31.33
12309-3
Hanes Magnet School
35.33
Page 1 of 1
`
	errs := verifyWithScores(text)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestParseTeamWithScoresParticipants(t *testing.T) {
	lines := cleanLines(`Participant ID
School
Division
Score
16897-2
Gaston Day School MS
31.33
12309-3
Hanes Magnet School
35.33
Page 1 of 1
`)
	participants, err := parseTeamWithScoresParticipants(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}

	p0 := participants[0]
	if p0.SchoolID != "16897" || p0.TeamNumber != "2" || p0.School != "Gaston Day School MS" {
		t.Errorf("unexpected first participant: %+v", p0)
	}

	p1 := participants[1]
	if p1.SchoolID != "12309" || p1.TeamNumber != "3" || p1.School != "Hanes Magnet School" {
		t.Errorf("unexpected second participant: %+v", p1)
	}
}
