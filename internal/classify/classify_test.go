package classify_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"expensetracker/internal/classify"
)

// testCategories is the candidate list every test in this file hands to
// Classify, standing in for an account's own expense categories.
var testCategories = []classify.Category{
	{ID: 7, Name: "Food & Drink"},
	{ID: 8, Name: "Transport"},
	{ID: 9, Name: "Other"},
}

// answerJSON builds a fake /v1/messages response whose single text block is
// the structured-output JSON {"category_id": id} -- the same shape a real
// Claude reply constrained by classify's schema would have.
func answerJSON(id int64) string {
	inner := fmt.Sprintf(`{"category_id":%d}`, id)
	encoded, _ := json.Marshal(inner)
	return fmt.Sprintf(`{
		"id": "msg_test",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-5",
		"stop_reason": "end_turn",
		"stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 5},
		"content": [{"type": "text", "text": %s}]
	}`, encoded)
}

// errorJSON builds the {"error": {...}} envelope the Anthropic API sends
// alongside a non-2xx status.
func errorJSON(errType, message string) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":%q,"message":%q}}`, errType, message)
}

// messageJSON builds a fake /v1/messages response with a caller-supplied
// raw "content" array, for tests that need a shape answerJSON can't
// produce: no content at all, or a block that isn't a text block.
func messageJSON(rawContent string) string {
	return fmt.Sprintf(`{
		"id": "msg_test",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-5",
		"stop_reason": "end_turn",
		"stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 5},
		"content": %s
	}`, rawContent)
}

func TestClassifyRequestCarriesModelEffortAndFormat(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(7))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key", Model: "claude-opus-5"}, server.URL)
	if _, err := c.Classify(context.Background(), "GRAB thanh toan chuyen di", testCategories); err != nil {
		t.Fatalf("Classify() error = %v, want nil", err)
	}

	if got := gotBody["model"]; got != "claude-opus-5" {
		t.Errorf("request model = %v, want %q", got, "claude-opus-5")
	}
	// max_tokens is the actual dollar cost of every call on this path; a
	// silent change to it (raised or dropped) would otherwise go
	// unnoticed by this test even though it exists specifically to pin
	// the request shape.
	if got := gotBody["max_tokens"]; got != float64(256) {
		t.Errorf("request max_tokens = %v, want %d", got, 256)
	}
	outputConfig, ok := gotBody["output_config"].(map[string]any)
	if !ok {
		t.Fatalf("request output_config = %v, want a map", gotBody["output_config"])
	}
	if got := outputConfig["effort"]; got != "low" {
		t.Errorf("request output_config.effort = %v, want %q", got, "low")
	}
	format, ok := outputConfig["format"].(map[string]any)
	if !ok {
		t.Fatalf("request output_config.format = %v, want a map", outputConfig["format"])
	}
	if got := format["type"]; got != "json_schema" {
		t.Errorf("request output_config.format.type = %v, want %q", got, "json_schema")
	}
	schema, ok := format["schema"].(map[string]any)
	if !ok {
		t.Fatalf("request output_config.format.schema = %v, want a map", format["schema"])
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("request schema.properties = %v, want a map", schema["properties"])
	}
	categoryID, ok := props["category_id"].(map[string]any)
	if !ok {
		t.Fatalf("request schema.properties.category_id = %v, want a map", props["category_id"])
	}
	enum, ok := categoryID["enum"].([]any)
	if !ok || len(enum) != len(testCategories) {
		t.Errorf("request schema.properties.category_id.enum = %v, want one entry per candidate", categoryID["enum"])
	}
}

func TestClassifyMapsAValidAnswerToTheRightCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(8))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	got, err := c.Classify(context.Background(), "GRAB thanh toan chuyen di", testCategories)
	if err != nil {
		t.Fatalf("Classify() error = %v, want nil", err)
	}
	if got != 8 {
		t.Errorf("Classify(%q) = %d, want %d", "GRAB thanh toan chuyen di", got, 8)
	}
}

func TestClassifyFallsBackOn429WithoutPanicking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, errorJSON("rate_limit_error", "rate limited"))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	if _, err := c.Classify(context.Background(), "note", testCategories); err == nil {
		t.Error("Classify() error = nil, want an error on a 429 response")
	}
}

func TestClassifyFallsBackOn500WithoutPanicking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, errorJSON("api_error", "internal server error"))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	if _, err := c.Classify(context.Background(), "note", testCategories); err == nil {
		t.Error("Classify() error = nil, want an error on a 500 response")
	}
}

func TestClassifyFallsBackWhenAnswerIsNotACandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 999 is not one of testCategories' ids -- a schema-enum violation
		// a well-behaved model would never produce, but Classify must not
		// trust the wire format blindly.
		fmt.Fprint(w, answerJSON(999))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	if _, err := c.Classify(context.Background(), "note", testCategories); err == nil {
		t.Error("Classify() error = nil, want an error when the answer names an id outside the candidate list")
	}
}

func TestClassifyFallsBackWhenAnswerBelongsToAnotherAccount(t *testing.T) {
	// Simulates a stale or cross-account id: 42 is a real category id
	// somewhere, just not among the candidates this call offered, so it
	// must be rejected exactly like an id that doesn't exist at all.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(42))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	if _, err := c.Classify(context.Background(), "note", testCategories); err == nil {
		t.Error("Classify() error = nil, want an error when the answer names a category not offered to this account")
	}
}

func TestClassifyUnconfiguredMakesNoCallAndFallsBack(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(7))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{}, server.URL)
	if c.Configured() {
		t.Fatal("Configured() = true for a Config with no APIKey, want false")
	}
	_, err := c.Classify(context.Background(), "note", testCategories)
	if !errors.Is(err, classify.ErrNotConfigured) {
		t.Errorf("Classify() error = %v, want errors.Is(err, ErrNotConfigured)", err)
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Errorf("server received %d requests, want 0 -- an unconfigured Classifier must never call out", got)
	}
}

// TestClassifyFallsBackOnAMalformedResponse covers the branch that decides
// whether a reply Classify cannot make sense of ever reaches the caller as
// something that looks like a real answer. This is the same family of bug
// slice 3 shipped as a Critical finding right after being called "safe by
// inspection" -- so each shape here is exercised rather than reasoned
// about. Every case must return a non-nil error and the zero id, never a
// panic and never an id a caller could mistake for a real category.
func TestClassifyFallsBackOnAMalformedResponse(t *testing.T) {
	tests := map[string]string{
		"no content blocks at all":           messageJSON(`[]`),
		"only a non-text block":              messageJSON(`[{"type":"thinking","thinking":"pondering which category fits"}]`),
		"text block that is not JSON":        messageJSON(`[{"type":"text","text":"sorry, I can't help with that"}]`),
		"text block that is an empty string": messageJSON(`[{"type":"text","text":""}]`),
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, body)
			}))
			defer server.Close()

			c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
			got, err := c.Classify(context.Background(), "note", testCategories)
			if err == nil {
				t.Fatalf("Classify() error = nil, want an error for a malformed response")
			}
			if got != 0 {
				t.Errorf("Classify() id = %d, want 0 alongside a non-nil error -- a nonzero id here could be mistaken for a real category by a careless caller", got)
			}
		})
	}
}

func TestClassifyWithNoCandidatesFallsBackWithoutCallingOut(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(7))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	_, err := c.Classify(context.Background(), "note", nil)
	if !errors.Is(err, classify.ErrNoCandidates) {
		t.Errorf("Classify() error = %v, want errors.Is(err, ErrNoCandidates)", err)
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Errorf("server received %d requests, want 0", got)
	}
}
