// Command survey performs offline branch validation and response
// interpretation for a JSON survey definition.
//
// Usage:
//
//	survey -input survey.json -outdir results
//
// On any definition error it exits with a non-zero status and writes no
// result file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"example.com/survey-branch-interpreter/internal/survey"
)

func main() {
	input := flag.String("input", "", "path to the survey JSON file")
	outDir := flag.String("outdir", "", "directory where results.json is written")
	flag.Parse()

	if *input == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: survey -input survey.json -outdir results")
		os.Exit(2)
	}
	if err := run(*input, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(input, outDir string) error {
	s, err := survey.LoadSurvey(input)
	if err != nil {
		return err
	}
	if err := s.Validate(); err != nil {
		return err
	}
	if err := s.ValidateGraph(); err != nil {
		return err
	}

	results, err := s.InterpretAll()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	out := survey.ResultFile{Entry: s.Entry, Responses: results}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	outPath := filepath.Join(outDir, "results.json")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d response(s))\n", outPath, len(results))
	return nil
}
