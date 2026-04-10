# nctsa-export-parser

A Go library for parsing NCTSA (North Carolina Technology Student Association) competition results PDFs into structured data.

## Install

```bash
go get github.com/nctsacomms/nctsa-export-parser
```

## Quick Start

```go
import parser "github.com/nctsacomms/nctsa-export-parser"

// Verify a PDF is in the expected format
if issues := parser.Verify("results.pdf"); issues != nil {
    log.Fatalf("invalid PDF: %v", issues)
}

// Parse into structured data
result, err := parser.Parse("results.pdf")
if err != nil {
    log.Fatal(err)
}

switch result.EventType {
case parser.TeamEvent:
    for _, p := range result.Team.Participants {
        fmt.Printf("School %s, Team %s: %s\n", p.SchoolID, p.TeamNumber, p.School)
    }
case parser.IndividualEvent:
    for _, p := range result.Individual.Participants {
        fmt.Printf("ID %s: %s\n", p.ID, p.School)
    }
}
```

## Supported Formats

- **Team events** -- IDs like `21333-1` (school ID + team number)
- **Individual events** -- IDs like `26432007` (8-digit participant ID)

Both formats must include a title with parenthesized abbreviation, a result type subtitle (e.g. Semi-Finalists), table headers (Participant ID, School, Division), and a page footer.

## Testing

```bash
go test -v -race ./...
```
