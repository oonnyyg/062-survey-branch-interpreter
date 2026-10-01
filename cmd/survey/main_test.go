package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDoc = `{
  "entry": "q1",
  "questions": [
    {"id": "q1", "type": "single",
     "options": [{"id": "a", "text": "A", "next": "END"}],
     "default_next": "END"}
  ],
  "responses": [{"id": "r1", "answers": {"q1": "a"}}]
}`

func writeInput(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "survey.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunWritesResults(t *testing.T) {
	input := writeInput(t, validDoc)
	outDir := filepath.Join(t.TempDir(), "out")
	if err := run(input, outDir); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "results.json"))
	if err != nil {
		t.Fatalf("results.json not written: %v", err)
	}
	if !strings.Contains(string(data), `"status": "ok"`) {
		t.Fatalf("unexpected output: %s", data)
	}
}

func TestRunRejectsBadDefinitionWithoutOutput(t *testing.T) {
	bad := strings.Replace(validDoc, `"next": "END"`, `"next": "GHOST"`, 1)
	input := writeInput(t, bad)
	outDir := filepath.Join(t.TempDir(), "out")
	err := run(input, outDir)
	if err == nil {
		t.Fatal("expected definition error")
	}
	if !strings.Contains(err.Error(), "unknown jump target") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "results.json")); !os.IsNotExist(statErr) {
		t.Fatalf("results.json must not be written on definition error, stat=%v", statErr)
	}
}
