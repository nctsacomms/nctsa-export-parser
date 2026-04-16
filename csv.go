package parser

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// CSVResult holds parsed data from a CSV export.
type CSVResult struct {
	Entries []CSVEntry
}

// CSVEntry represents a single row from a CSV export.
// Score is deliberately excluded from output.
type CSVEntry struct {
	ID            string
	ChapterName   string
	ChapterNumber string
}

var csvExpectedHeaders = []string{"ID", "ChapterName", "ChapterNumber", "Score"}

func parseCSVFile(path string) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("opening CSV file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return Result{}, fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) == 0 {
		return Result{}, fmt.Errorf("CSV file is empty")
	}

	headers := records[0]
	if len(headers) < 3 {
		return Result{}, fmt.Errorf("CSV has %d columns, expected at least 3", len(headers))
	}
	for i, expected := range csvExpectedHeaders[:3] {
		if strings.TrimSpace(headers[i]) != expected {
			return Result{}, fmt.Errorf("CSV header mismatch: column %d is %q, expected %q", i, headers[i], expected)
		}
	}

	var entries []CSVEntry
	for _, row := range records[1:] {
		if len(row) < 3 {
			continue
		}
		entries = append(entries, CSVEntry{
			ID:            row[0],
			ChapterName:   row[1],
			ChapterNumber: row[2],
		})
	}

	if len(entries) == 0 {
		return Result{}, fmt.Errorf("CSV contains no data rows")
	}

	return Result{
		FileType: FileCSV,
		CSV:      &CSVResult{Entries: entries},
	}, nil
}
