package survey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadSurvey reads and parses a survey definition file. JSON syntax errors
// are reported with the decoder offset to help locate the problem.
func LoadSurvey(path string) (*Survey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Survey
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("invalid JSON: unexpected trailing content")
	}
	return &s, nil
}
