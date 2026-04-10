package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	parser "github.com/nctsacomms/nctsa-export-parser"
)

func main() {
	inputDir := "raw_data"
	dirs := map[parser.EventType]string{
		parser.TeamEvent:       "team_data",
		parser.IndividualEvent: "individual_data",
	}
	unparsedDir := "unparsed"

	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}
	os.MkdirAll(unparsedDir, 0o755)

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputDir, err)
		os.Exit(1)
	}

	var team, individual, unparsed, skipped int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		src := filepath.Join(inputDir, name)

		if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
			fmt.Printf("SKIP  %s\n", name)
			skipped++
			continue
		}

		issues := parser.Verify(src)
		if issues != nil {
			fmt.Printf("COPY  %s -> unparsed/ (%s)\n", name, issues[0])
			copyFile(src, filepath.Join(unparsedDir, name))
			unparsed++
			continue
		}

		result, err := parser.Parse(src)
		if err != nil {
			fmt.Printf("COPY  %s -> unparsed/ (%v)\n", name, err)
			copyFile(src, filepath.Join(unparsedDir, name))
			unparsed++
			continue
		}

		destDir := dirs[result.EventType]
		fmt.Printf("SORT  %s -> %s/\n", name, destDir)
		copyFile(src, filepath.Join(destDir, name))

		switch result.EventType {
		case parser.TeamEvent:
			team++
		case parser.IndividualEvent:
			individual++
		}
	}

	fmt.Printf("\nDone: %d team, %d individual, %d unparsed, %d skipped\n", team, individual, unparsed, skipped)
}

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer out.Close()

	io.Copy(out, in)
}
