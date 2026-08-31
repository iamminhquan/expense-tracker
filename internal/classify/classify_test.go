package classify_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// answerJSON builds a fake generateContent response whose single text part
// is the structured-output JSON {"category_id":"id"} -- the same shape a
// real Gemini reply constrained by classify's schema would have. The id is
// a string because the schema constrains it with a STRING enum, which is
// the only type Gemini documents enum for.
func answerJSON(id int64) string {
	return candidateJSON(fmt.Sprintf(`[{"text":%q}]`, fmt.Sprintf(`{"category_id":"%d"}`, id)), "STOP")
}

// candidateJSON builds a generateContent response with a caller-supplied
// raw "parts" array and finish reason, for tests that need a shape
// answerJSON can't produce: no parts at all, a truncated answer, or text
// that isn't JSON.
func candidateJSON(rawParts, finishReason string) string {
	return fmt.Sprintf(`{
		"candidates": [{
			"content": {"role": "model", "parts": %s},
			"finishReason": %q
		}],
		"usageMetadata": {"promptTokenCount": 120, "candidatesTokenCount": 8},
		"modelVersion": "gemini-3.5-flash-lite"
	}`, rawParts, finishReason)
}

// errorJSON builds the {"error": {...}} envelope Google APIs send alongside
// a non-2xx status.
func errorJSON(code int, status, message string) string {
	return fmt.Sprintf(`{"error":{"code":%d,"message":%q,"status":%q}}`, code, message, status)
}

func TestClassifyRequestCarriesModelKeyAndSchema(t *testing.T) {
	var gotBody map[string]any
	var gotPath, gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-goog-api-key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(7))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key", Model: "gemini-3.5-flash-lite"}, server.URL)
	if _, err := c.Classify(context.Background(), "GRAB thanh toan chuyen di", testCategories); err != nil {
		t.Fatalf("Classify() error = %v, want nil", err)
	}

	// Gemini names the model in the URL, not the body, so a model that
	// silently stopped being sent would look identical in the payload.
	if want := "/models/gemini-3.5-flash-lite:generateContent"; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
	// The key must ride in the header, never the query string, so it stays
	// out of proxy logs.
	if gotKey != "test-key" {
		t.Errorf("x-goog-api-key header = %q, want %q", gotKey, "test-key")
	}

	cfg, ok := gotBody["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("request generationConfig = %v, want a map", gotBody["generationConfig"])
	}
	if got := cfg["responseMimeType"]; got != "application/json" {
		t.Errorf("request responseMimeType = %v, want %q", got, "application/json")
	}
	// maxOutputTokens has to leave room for a thinking model to reason
	// before it answers; sized too tightly the call comes back MAX_TOKENS
	// with no answer at all, which no other test here would catch.
	if got := cfg["maxOutputTokens"]; got != float64(2048) {
		t.Errorf("request maxOutputTokens = %v, want %d", got, 2048)
	}
	if got := cfg["temperature"]; got != float64(0) {
		t.Errorf("request temperature = %v, want 0", got)
	}
	// thinkingConfig is deliberately never sent: the API errors on it for
	// models that don't support thinking, and GEMINI_MODEL is meant to be
	// changeable without editing classify.go.
	if _, present := cfg["thinkingConfig"]; present {
		t.Errorf("request carried thinkingConfig = %v, want it absent", cfg["thinkingConfig"])
	}

	schema, ok := cfg["responseSchema"].(map[string]any)
	if !ok {
		t.Fatalf("request responseSchema = %v, want a map", cfg["responseSchema"])
	}
	// Gemini's schema is the OpenAPI subset: uppercase type names, and no
	// additionalProperties (which it rejects outright).
	if got := schema["type"]; got != "OBJECT" {
		t.Errorf("request responseSchema.type = %v, want %q", got, "OBJECT")
	}
	if _, present := schema["additionalProperties"]; present {
		t.Error("request responseSchema carried additionalProperties, which the Gemini API does not accept")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("request responseSchema.properties = %v, want a map", schema["properties"])
	}
	categoryID, ok := props["category_id"].(map[string]any)
	if !ok {
		t.Fatalf("request responseSchema.properties.category_id = %v, want a map", props["category_id"])
	}
	if got := categoryID["type"]; got != "STRING" {
		t.Errorf("request category_id.type = %v, want %q -- Gemini documents enum only for Type.STRING", got, "STRING")
	}
	enum, ok := categoryID["enum"].([]any)
	if !ok || len(enum) != len(testCategories) {
		t.Fatalf("request category_id.enum = %v, want one entry per candidate", categoryID["enum"])
	}
	for i, want := range []string{"7", "8", "9"} {
		if enum[i] != want {
			t.Errorf("request category_id.enum[%d] = %v, want %q", i, enum[i], want)
		}
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

// TestClassifyUsesTheDefaultModelWhenConfigLeavesItEmpty pins the fallback
// that only fires for a Config built by hand: internal/config.Load applies
// the same default, so nothing else in the app exercises this path.
func TestClassifyUsesTheDefaultModelWhenConfigLeavesItEmpty(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, answerJSON(7))
	}))
	defer server.Close()

	c := classify.NewWithEndpoint(classify.Config{APIKey: "test-key"}, server.URL)
	if _, err := c.Classify(context.Background(), "note", testCategories); err != nil {
		t.Fatalf("Classify() error = %v, want nil", err)
	}
	if !strings.HasPrefix(gotPath, "/models/gemini-") || !strings.HasSuffix(gotPath, ":generateContent") {
		t.Errorf("request path = %q, want a /models/gemini-...:generateContent path", gotPath)
	}
}

func TestClassifyFallsBackOn429WithoutPanicking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, errorJSON(429, "RESOURCE_EXHAUSTED", "Quota exceeded for quota metric"))
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
		fmt.Fprint(w, errorJSON(500, "INTERNAL", "internal error"))
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
		"no candidates at all":              `{"candidates":[]}`,
		"candidate with no parts":           candidateJSON(`[]`, "STOP"),
		"truncated before writing any":      candidateJSON(`[]`, "MAX_TOKENS"),
		"prompt blocked by a safety rule":   `{"promptFeedback":{"blockReason":"SAFETY"}}`,
		"text part that is not JSON":        candidateJSON(`[{"text":"sorry, I can't help with that"}]`, "STOP"),
		"text part that is an empty string": candidateJSON(`[{"text":""}]`, "STOP"),
		"category_id that is not a number":  candidateJSON(`[{"text":"{\"category_id\":\"Food \\u0026 Drink\"}"}]`, "STOP"),
		"body that is not JSON at all":      `<html>502 Bad Gateway</html>`,
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
