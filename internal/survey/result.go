package survey

// Step records one visited question and why the interpreter jumped onward.
// On an answer error the failing question is the last step and next is empty.
type Step struct {
	QuestionID string `json:"question_id"`
	Answer     any    `json:"answer"`
	Basis      string `json:"basis"`
	Next       string `json:"next,omitempty"`
}

// ResponseResult is the interpretation outcome for one response.
type ResponseResult struct {
	ResponseID         string   `json:"response_id"`
	Status             string   `json:"status"`
	Error              string   `json:"error,omitempty"`
	ErrorQuestionID    string   `json:"error_question_id,omitempty"`
	Path               []Step   `json:"path"`
	UnusedAnswers      []string `json:"unused_answers"`
	UnvisitedQuestions []string `json:"unvisited_questions"`
}

// ResultFile is the top-level output document, sorted by response id.
type ResultFile struct {
	Entry     string           `json:"entry"`
	Responses []ResponseResult `json:"responses"`
}
