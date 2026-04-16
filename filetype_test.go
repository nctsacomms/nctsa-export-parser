package parser

import "testing"

func TestClassifyText_TeamSemiFinalist(t *testing.T) {
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
	got := classifyText(text)
	if got != FileTeamSemiFinalist {
		t.Errorf("expected FileTeamSemiFinalist, got %s", got)
	}
}

func TestClassifyText_IndividualSemiFinalist(t *testing.T) {
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
	got := classifyText(text)
	if got != FileIndividualSemiFinalist {
		t.Errorf("expected FileIndividualSemiFinalist, got %s", got)
	}
}

func TestClassifyText_SemiFinalistWithScores(t *testing.T) {
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
	got := classifyText(text)
	if got != FileSemiFinalistWithScores {
		t.Errorf("expected FileSemiFinalistWithScores, got %s", got)
	}
}

func TestClassifyText_ScheduleClean(t *testing.T) {
	text := `HS Chapter Team - Finalist Presentations
4/10/2026 Lvl: 2
3DUWLFLSDQW
,'
Christian Burgess
Phillip O. Berry Academy Of Technology
21473022
21473-1
1
9:00
AM
`
	got := classifyText(text)
	if got != FileScheduleClean {
		t.Errorf("expected FileScheduleClean, got %s", got)
	}
}

func TestClassifyText_ScheduleRaw(t *testing.T) {
	text := `0667(0$QLPDWLRQ/HYHO
,QGLYLGXDO6FKHGXOHV
Displays all individual and team schedules for this event.
Schedule Name: Finalist Interviews
Schedule Date: 4/9/2026
Schedule Time Range: 3:00 PM to 6:00 PM
Swap
Insert Break
`
	got := classifyText(text)
	if got != FileScheduleRaw {
		t.Errorf("expected FileScheduleRaw, got %s", got)
	}
}

func TestClassifyText_Empty(t *testing.T) {
	got := classifyText("")
	if got != FileUnknown {
		t.Errorf("expected FileUnknown, got %s", got)
	}
}
