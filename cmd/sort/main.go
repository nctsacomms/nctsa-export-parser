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
	dirs := map[parser.FileType]string{
		parser.FileTeamSemiFinalist:       "team_data",
		parser.FileIndividualSemiFinalist: "individual_data",
		parser.FileSemiFinalistWithScores: "team_data",
		parser.FileScheduleClean:          "schedule_data",
		parser.FileScheduleRaw:            "schedule_data",
		parser.FileCSV:                    "csv_data",
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

	counts := make(map[parser.FileType]int)
	var unparsed, skipped int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		src := filepath.Join(inputDir, name)
		ext := strings.ToLower(filepath.Ext(name))

		if ext != ".pdf" && ext != ".csv" {
			fmt.Printf("SKIP  %s\n", name)
			skipped++
			continue
		}

		ft, err := parser.Classify(src)
		if err != nil || ft == parser.FileUnknown {
			reason := "unknown format"
			if err != nil {
				reason = err.Error()
			}
			fmt.Printf("COPY  %s -> unparsed/ (%s)\n", name, reason)
			copyFile(src, filepath.Join(unparsedDir, name))
			unparsed++
			continue
		}

		_, err = parser.Parse(src)
		if err != nil {
			fmt.Printf("COPY  %s -> unparsed/ (%v)\n", name, err)
			copyFile(src, filepath.Join(unparsedDir, name))
			unparsed++
			continue
		}

		destDir := dirs[ft]
		fmt.Printf("SORT  %-15s %s -> %s/\n", "["+string(ft)+"]", name, destDir)
		copyFile(src, filepath.Join(destDir, name))
		counts[ft]++
	}

	fmt.Println()
	fmt.Println("Summary:")
	for ft, count := range counts {
		fmt.Printf("  %-30s %d\n", ft, count)
	}
	fmt.Printf("  %-30s %d\n", "unparsed", unparsed)
	fmt.Printf("  %-30s %d\n", "skipped", skipped)
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
