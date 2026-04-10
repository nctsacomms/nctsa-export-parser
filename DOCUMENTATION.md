# nctsa-export-parser -- Agent Documentation

Detailed reference for AI agents working with this package.

## Package Overview

**Import path:** `github.com/nctsacomms/nctsa-export-parser`

**Package name:** `parser`

**Source file:** `parser.go` (single file, ~285 lines)

**Dependency:** `github.com/ledongthuc/pdf` for PDF text extraction

This library parses NCTSA competition results PDFs exported from `registermychapter.com` into Go structs. It handles two event formats (team and individual) distinguished by participant ID structure.

## Exported API

### Functions

#### `Parse(pdfPath string) (Result, error)`

Primary entry point. Reads a PDF file, verifies it matches the expected format, classifies the event type, and returns structured participant data.

Internally calls: `extractText` -> `verify` -> `classifyEvent` -> `parseTeamParticipants` or `parseIndividualParticipants`.

Returns an error if:
- The file cannot be opened or read
- Format verification fails (wraps all issues into a single error string)
- No participant rows are found after parsing

#### `Verify(pdfPath string) []string`

Validation-only entry point. Returns `nil` if the PDF matches the expected format, or a slice of human-readable issue strings. Use this to check a PDF without parsing it.

**Verification checks (in order):**
1. Title contains a parenthesized event abbreviation, e.g. `(HS)`
2. Subtitle contains a result type keyword: `FINALISTS`, `SEMI-FINALISTS`, `WINNERS`, `RESULTS`, or `QUALIFIERS` (handles small-caps text fragmentation by stripping whitespace before matching)
3. Table headers `Participant ID`, `School`, and `Division` each appear as exact lines
4. At least one participant ID row exists (team or individual format)
5. Each participant ID is followed by a school name line (not another ID)
6. Page footer matches `Page X of Y`

### Types

#### `EventType`

```go
type EventType string

const (
    TeamEvent       EventType = "team"
    IndividualEvent EventType = "individual"
)
```

#### `Result`

```go
type Result struct {
    EventType  EventType
    Team       *TeamResult       // non-nil when EventType == TeamEvent
    Individual *IndividualResult // non-nil when EventType == IndividualEvent
}
```

Tagged union. Exactly one of `Team` or `Individual` is non-nil based on `EventType`. Always check `EventType` or nil-check before accessing.

#### `TeamResult`

```go
type TeamResult struct {
    Participants []TeamParticipant
}
```

#### `TeamParticipant`

```go
type TeamParticipant struct {
    SchoolID   string // 5-digit school identifier, e.g. "21333"
    TeamNumber string // team number within the school, e.g. "1"
    School     string // school name, e.g. "Panther Creek High School"
}
```

Parsed from IDs matching `^\d{5}-\d+$`. The ID `21333-1` becomes `SchoolID: "21333", TeamNumber: "1"`.

#### `IndividualResult`

```go
type IndividualResult struct {
    Participants []IndividualParticipant
}
```

#### `IndividualParticipant`

```go
type IndividualParticipant struct {
    ID     string // 7-8 digit participant ID, e.g. "26432007"
    School string // school name, e.g. "Woodlawn School"
}
```

Parsed from IDs matching `^\d{7,8}$`.

## PDF Format Specification

The parser expects PDFs exported from `registermychapter.com` with this structure:

```
<Event Name> (<ABBREVIATION>)          <- title with parenthesized code
<Result Type> - in Random Order        <- e.g. "Semi-Finalists"
Participant ID | School | Division     <- table headers (rendered as separate lines)
<ID>                                   <- participant ID
<School Name>                          <- school name on next line
...                                    <- repeating ID/school pairs
Page X of Y                           <- footer
```

**Important:** The PDFs use small-caps rendering which causes text extraction to fragment words across lines (e.g. `"S\nEMI\n-F\nINALISTS"`). The verify function handles this by compacting whitespace before keyword matching.

### ID Format Rules

| Event Type | Pattern | Example | Regex |
|---|---|---|---|
| Team | 5 digits, hyphen, number | `21333-1` | `^\d{5}-\d+$` |
| Individual | 7-8 digits | `26432007` | `^\d{7,8}$` |

Classification is determined by the first ID encountered in the document. The combined pattern used for general matching is `^(\d{5}-\d+\|\d{7,8})$`.

## Internal Functions (unexported)

These are not callable from outside the package but are relevant for understanding the code:

| Function | Purpose |
|---|---|
| `extractText(pdfPath)` | Opens PDF, concatenates plain text from all pages |
| `verify(text)` | Runs 6 structural checks on extracted text |
| `classifyEvent(text)` | Returns `TeamEvent` or `IndividualEvent` based on first ID found |
| `parseTeamParticipants(lines)` | Extracts `[]TeamParticipant` from cleaned lines |
| `parseIndividualParticipants(lines)` | Extracts `[]IndividualParticipant` from cleaned lines |
| `cleanLines(text)` | Splits on newlines, trims whitespace, removes empty lines |

## Usage Patterns

### Basic parse with type switch

```go
result, err := parser.Parse("event.pdf")
if err != nil {
    return err
}

switch result.EventType {
case parser.TeamEvent:
    for _, p := range result.Team.Participants {
        fmt.Printf("%s-%s %s\n", p.SchoolID, p.TeamNumber, p.School)
    }
case parser.IndividualEvent:
    for _, p := range result.Individual.Participants {
        fmt.Printf("%s %s\n", p.ID, p.School)
    }
}
```

### Verify before parsing

```go
if issues := parser.Verify(path); issues != nil {
    for _, issue := range issues {
        log.Printf("  - %s", issue)
    }
    return fmt.Errorf("invalid PDF format")
}
// Verify is called again inside Parse, so this is only useful
// when you want to inspect issues without parsing.
```

### Batch processing a directory

```go
entries, _ := os.ReadDir("pdfs/")
for _, e := range entries {
    if !strings.HasSuffix(e.Name(), ".pdf") {
        continue
    }
    result, err := parser.Parse(filepath.Join("pdfs/", e.Name()))
    if err != nil {
        log.Printf("skipping %s: %v", e.Name(), err)
        continue
    }
    // process result...
}
```

## Testing

```bash
go test -v -race ./...
```

Test file: `parser_test.go` (13 tests)

Tests cover: valid team text, valid individual text, missing title, missing result type, missing headers, no data rows, missing school after ID, missing page footer, empty text, event classification (team/individual), and participant parsing (team/individual).

All tests operate on raw text strings passed to unexported functions -- no PDF fixtures are required.
