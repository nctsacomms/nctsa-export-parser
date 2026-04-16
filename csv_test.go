package parser

import (
	"os"
	"testing"
)

func TestParseCSVFile_Valid(t *testing.T) {
	content := `"ID","ChapterName","ChapterNumber","Score"
"25764-2","Marvin Ridge High Schooll","5764",null
"26319-2","Triangle Math And Science Academy","6319",null
"21023-1","NC School Of Science & Math","1023",null
`
	f := writeTempFile(t, content, ".csv")

	result, err := parseCSVFile(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FileType != FileCSV {
		t.Errorf("expected FileCSV, got %s", result.FileType)
	}
	if result.CSV == nil {
		t.Fatal("CSV result is nil")
	}
	if len(result.CSV.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result.CSV.Entries))
	}

	e := result.CSV.Entries[0]
	if e.ID != "25764-2" || e.ChapterName != "Marvin Ridge High Schooll" || e.ChapterNumber != "5764" {
		t.Errorf("unexpected first entry: %+v", e)
	}
}

func TestParseCSVFile_Empty(t *testing.T) {
	f := writeTempFile(t, "", ".csv")

	_, err := parseCSVFile(f)
	if err == nil {
		t.Error("expected error for empty CSV")
	}
}

func TestParseCSVFile_BadHeaders(t *testing.T) {
	content := `"Wrong","Headers","Here"
"a","b","c"
`
	f := writeTempFile(t, content, ".csv")

	_, err := parseCSVFile(f)
	if err == nil {
		t.Error("expected error for bad headers")
	}
}

func TestParseCSVFile_NoDataRows(t *testing.T) {
	content := `"ID","ChapterName","ChapterNumber","Score"
`
	f := writeTempFile(t, content, ".csv")

	_, err := parseCSVFile(f)
	if err == nil {
		t.Error("expected error for no data rows")
	}
}

func writeTempFile(t *testing.T, content, suffix string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test*"+suffix)
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
