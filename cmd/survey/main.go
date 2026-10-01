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
	input := flag.String("input", "", "path to the survey definition JSON file")
	outDir := flag.String("out", "", "directory for the output results JSON")
	flag.Parse()
	if *input == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: survey -input <definition.json> -out <output-dir>")
		os.Exit(2)
	}
	if err := run(*input, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(input, outDir string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	def, err := survey.ParseDefinition(data)
	if err != nil {
		return fmt.Errorf("invalid survey definition: %w", err)
	}
	results := def.InterpretAll()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	out, err := json.MarshalIndent(map[string]any{"results": results}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode results: %w", err)
	}
	outPath := filepath.Join(outDir, "results.json")
	if err := os.WriteFile(outPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	fmt.Println("wrote", outPath)
	return nil
}
