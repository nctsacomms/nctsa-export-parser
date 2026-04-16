package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// FileType identifies the source format of a file.
type FileType string

const (
	FileTeamSemiFinalist       FileType = "team_semi_finalist"
	FileIndividualSemiFinalist FileType = "individual_semi_finalist"
	FileSemiFinalistWithScores FileType = "semi_finalist_with_scores"
	FileScheduleClean          FileType = "schedule_clean"
	FileScheduleRaw            FileType = "schedule_raw"
	FileCSV                    FileType = "csv"
	FileUnknown                FileType = "unknown"
)

var scheduleLvlPattern = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{4}\s+Lvl:\s*\d+`)

// Classify determines the FileType of a given file path.
func Classify(path string) (FileType, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".csv" {
		return FileCSV, nil
	}

	if ext != ".pdf" {
		return FileUnknown, nil
	}

	text, err := extractText(path)
	if err != nil {
		return FileUnknown, fmt.Errorf("extracting text: %w", err)
	}

	return classifyText(text), nil
}

// classifyText determines the FileType from extracted PDF text.
func classifyText(text string) FileType {
	lines := cleanLines(text)
	if len(lines) == 0 {
		return FileUnknown
	}

	// Check for Schedule Raw (Format B): contains metadata lines
	for _, line := range lines {
		if strings.HasPrefix(line, "Schedule Date:") || strings.HasPrefix(line, "Schedule Time Range:") {
			return FileScheduleRaw
		}
	}

	// Check for Schedule Clean (Format A): second line has date + "Lvl:"
	if len(lines) >= 2 && scheduleLvlPattern.MatchString(lines[1]) {
		return FileScheduleClean
	}

	// Check for Semi-Finalist with Scores: has "Score" header
	hasScoreHeader := false
	for _, line := range lines {
		if line == "Score" {
			hasScoreHeader = true
			break
		}
	}
	if hasScoreHeader {
		// Confirm it also has semi-finalist markers
		joined := strings.Join(lines, " ")
		upper := strings.ToUpper(joined)
		if titleAbbrevPattern.MatchString(upper) {
			return FileSemiFinalistWithScores
		}
	}

	// Standard semi-finalist: classify by first ID
	for _, line := range lines {
		if teamIDPattern.MatchString(line) {
			return FileTeamSemiFinalist
		}
		if individualIDPattern.MatchString(line) {
			return FileIndividualSemiFinalist
		}
	}

	return FileUnknown
}
