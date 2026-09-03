// Package classify asks Gemini to pick a category for a bank-transfer note,
// choosing only among the categories the caller says an account is allowed
// to use.
//
// Shaped like internal/mailer on purpose, down to calling the provider's
// HTTP API directly rather than through an SDK: a Config carrying an
// overridable endpoint (which is what lets a test point Classify at an
// httptest server instead of the real API), a Configured check, and a call
// that takes ctx first. One JSON POST is the whole integration, so an SDK
// would add a dependency and a release cadence to track without removing any
// code worth removing.
//
// Callers must treat every error this package returns -- not configured, a
// network failure, a 429/500, a malformed or out-of-candidate answer -- as
// equivalent to "could not classify" and fall back accordingly. See
// internal/inboxproc's resolveCategoryForNotice for why: a
// classification failure is allowed to cost a wrong category, never a whole
// transaction.
package classify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultEndpoint is the Gemini API base the model path is appended to.
// Overridable through Config.Endpoint so tests never reach the network.
const defaultEndpoint = "https://generativelanguage.googleapis.com/v1beta"

// defaultModel is used when Config.Model is empty. internal/config.Load
// already applies this same default to GEMINI_MODEL, so in practice
// Classify only falls back to it when a test builds a Config by hand.
//
// A "flash-lite" model is the right size for this: the task is picking one
// id out of a short list, which is the cheapest thing an LLM can be asked
// to do, and it is the tier Google's free quota is most generous with.
const defaultModel = "gemini-3.5-flash-lite"

// classifyTimeout bounds a single Classify call. This is what keeps a hung
// or merely slow Gemini request from stalling processPendingEmails's
// goroutine indefinitely -- the background loop has no other timeout of its
// own.
const classifyTimeout = 20 * time.Second

// maxOutputTokens is far larger than the one-line JSON answer needs,
// because on a thinking-capable model the budget is spent on reasoning
// tokens first and the answer is written from what is left. Sized tightly
// (256 was enough under the previous provider, where thinking was
// configured separately) a request can come back with finishReason
// MAX_TOKENS and no answer at all.
//
// The alternative -- sending generationConfig.thinkingConfig to turn
// thinking down -- is deliberately not used: the API documents an error for
// that field on models that do not support thinking, and GEMINI_MODEL
// exists precisely so the model can be changed without touching this file.
// Paying for a few unused tokens of headroom is cheaper than a config that
// 400s the day someone picks a different model.
const maxOutputTokens = 2048

// ErrNotConfigured is returned by Classify without making any network call
// when Config.APIKey is empty. Callers use it (via errors.Is) to skip
// logging what would otherwise look like a failure -- an unset API key is
// the ordinary, expected state for an account that has never opted into
// the feature, the same way mailer.Mailer.Send treats a missing Brevo key.
var ErrNotConfigured = errors.New("classify: not configured")

// ErrNoCandidates is returned when the caller passes no categories to
// choose among. There is nothing useful to ask the model in that case, and
// asking anyway would spend a call on an answer that could never be valid.
var ErrNoCandidates = errors.New("classify: no candidate categories")

// Category is one of the categories the account is allowed to use: its
// i18n-resolved display name and the id that gets written back as a hint
// if the model picks it. The caller is responsible for filtering this list
// to the notice's own direction (income vs. expense) before calling
// Classify -- the model is never given a category it should not be able to
// choose.
type Category struct {
	ID   int64
	Name string
}

// Config holds the Gemini API settings. Both fields are optional, the same
// way internal/mailer.Config treats the Brevo settings: an empty APIKey
// just means Classify returns ErrNotConfigured (logged by the caller, not
// fatal) rather than the app refusing to start.
type Config struct {
	APIKey string
	Model  string
	// Endpoint overrides the Gemini API base, so a test can point Classify at
	// an httptest server instead of the real API. Empty means defaultEndpoint.
	Endpoint string
}

// Classifier calls Gemini to classify a note against a caller-supplied list
// of categories.
type Classifier struct {
	cfg      Config
	client   *http.Client
	endpoint string
}

// New constructs a Classifier that calls the Gemini API in cfg.
func New(cfg Config) *Classifier {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Classifier{cfg: cfg, client: &http.Client{}, endpoint: endpoint}
}

// Configured reports whether enough settings are present to attempt a call.
// Callers use this to skip the call (and log a clear "not configured"
// reason) instead of letting Classify fail with an opaque 401 against an
// empty API key.
func (c *Classifier) Configured() bool {
	return c.cfg.APIKey != ""
}

// Classify asks which of cats best fits note and returns that category's
// ID. It never returns an ID that was not in cats: the response schema
// constrains category_id to the exact set of candidate ids via an enum, and
// the answer is checked against cats again here as defense in depth against
// a bug in that constraint (or, in principle, the model ignoring it).
//
// Every failure -- not configured, no candidates, a network error, a
// non-2xx response, an answer that doesn't parse or names an id outside
// cats -- is returned as a plain error. Nothing here decides what a
// caller's fallback category is; that is resolveCategoryForNotice's job,
// deliberately kept out of this package so classify stays free of any
// notion of "Other".
func (c *Classifier) Classify(ctx context.Context, note string, cats []Category) (int64, error) {
	if !c.Configured() {
		return 0, ErrNotConfigured
	}
	if len(cats) == 0 {
		return 0, ErrNoCandidates
	}

	ctx, cancel := context.WithTimeout(ctx, classifyTimeout)
	defer cancel()

	body, err := c.post(ctx, generateRequest{
		Contents: []content{{
			Role:  "user",
			Parts: []part{{Text: buildPrompt(note, cats)}},
		}},
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   answerSchema(cats),
			MaxOutputTokens:  maxOutputTokens,
			// The task has one right answer given the list, so there is
			// nothing for sampling to add -- and a hint written back from a
			// coin-flip would be wrong every time it was reused.
			Temperature: 0,
		},
	})
	if err != nil {
		return 0, err
	}

	answered, err := answerText(body)
	if err != nil {
		return 0, err
	}

	var answer classifyAnswer
	if err := json.Unmarshal([]byte(answered), &answer); err != nil {
		return 0, fmt.Errorf("classify: parse response: %w", err)
	}

	// The schema constrains category_id to a STRING enum rather than an
	// integer one, because Gemini's schema documents enum only for
	// Type.STRING. The id is parsed back here; the enum still does the
	// constraining, this is just the type it had to travel as.
	id, err := strconv.ParseInt(strings.TrimSpace(answer.CategoryID), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("classify: category id %q is not a number: %w", answer.CategoryID, err)
	}

	for _, cand := range cats {
		if cand.ID == id {
			return id, nil
		}
	}
	return 0, fmt.Errorf("classify: model answered category %d, which is not one of the %d candidates offered", id, len(cats))
}

// post sends one generateContent request and returns the raw response body.
func (c *Classifier) post(ctx context.Context, payload generateRequest) ([]byte, error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("classify: encode request: %w", err)
	}

	model := c.cfg.Model
	if model == "" {
		model = defaultModel
	}
	url := fmt.Sprintf("%s/models/%s:generateContent", strings.TrimSuffix(c.endpoint, "/"), model)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("classify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// The key travels as a header rather than the ?key= query parameter the
	// API also accepts: a URL is the part of a request that ends up in
	// proxy logs and error messages, and a credential should not.
	req.Header.Set("x-goog-api-key", c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("classify: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("classify: read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, statusError(resp.StatusCode, respBody)
	}
	return respBody, nil
}

// buildPrompt writes the note and the caller's candidate categories as
// "id: name" lines. The model is told to choose only from this list --
// pairing that instruction with the enum in answerSchema is what makes
// "invent a category" not just discouraged but structurally impossible to
// answer with.
func buildPrompt(note string, cats []Category) string {
	var b strings.Builder
	b.WriteString("Classify this bank transfer note (it may be in Vietnamese) into exactly one of the categories listed below. Answer with that category's id. Do not invent a category that is not in the list.\n\n")
	b.WriteString("Note: ")
	b.WriteString(note)
	b.WriteString("\n\nCategories:\n")
	for _, cat := range cats {
		fmt.Fprintf(&b, "- id %d: %s\n", cat.ID, cat.Name)
	}
	return b.String()
}

// answerSchema builds the schema sent as generationConfig.responseSchema.
// Constraining category_id with an "enum" of the exact candidate ids -- not
// just a bare string -- is what makes the structured output do double duty
// as validation: a well-behaved model literally cannot express an id
// outside cats in its response.
//
// Two details are Gemini's rather than ours. Type names are the OpenAPI
// spelling ("OBJECT", "STRING"), not JSON Schema's lowercase; and the
// schema carries no "additionalProperties", which this API does not accept
// -- the required/enum pair is what pins the shape instead.
func answerSchema(cats []Category) map[string]any {
	ids := make([]any, len(cats))
	for i, cat := range cats {
		ids[i] = strconv.FormatInt(cat.ID, 10)
	}
	return map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"category_id": map[string]any{
				"type":        "STRING",
				"format":      "enum",
				"enum":        ids,
				"description": "id of the chosen category, one of the candidates given",
			},
		},
		"required":         []any{"category_id"},
		"propertyOrdering": []any{"category_id"},
	}
}

// answerText pulls the model's text out of a generateContent response,
// naming the two ways a 200 can still carry no answer -- a prompt blocked
// by a safety filter, and a response truncated before the model wrote one
// -- because both look identical to a caller that only checks for an empty
// string.
func answerText(body []byte) (string, error) {
	var resp generateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("classify: decode response: %w", err)
	}
	if resp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("classify: prompt blocked (%s)", resp.PromptFeedback.BlockReason)
	}
	if len(resp.Candidates) == 0 {
		return "", errors.New("classify: response carried no candidates")
	}

	cand := resp.Candidates[0]
	var b strings.Builder
	for _, p := range cand.Content.Parts {
		b.WriteString(p.Text)
	}
	if b.Len() == 0 {
		if cand.FinishReason != "" && cand.FinishReason != "STOP" {
			return "", fmt.Errorf("classify: no answer, finish reason %s", cand.FinishReason)
		}
		return "", errors.New("classify: response carried no text")
	}
	return b.String(), nil
}

// statusError turns a non-2xx response into an error carrying the API's own
// code and status string, which is what makes a 429 read differently from a
// 500 in a log line without the caller string-matching anything. Every status
// becomes a plain error a caller treats identically (fall back), so the code
// travels in the message rather than in a branch.
func statusError(code int, body []byte) error {
	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("classify: gemini api error (%d %s): %s", code, apiErr.Error.Status, apiErr.Error.Message)
	}
	return fmt.Errorf("classify: gemini api error (%d): %s", code, body)
}

// classifyAnswer is the shape the structured output is constrained to. A
// single required field, nothing else -- there is no prose to parse and no
// way for the model to answer with anything but a candidate category id.
type classifyAnswer struct {
	CategoryID string `json:"category_id"`
}

type generateRequest struct {
	Contents         []content        `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMIMEType string         `json:"responseMimeType"`
	ResponseSchema   map[string]any `json:"responseSchema"`
	MaxOutputTokens  int            `json:"maxOutputTokens"`
	Temperature      float64        `json:"temperature"`
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []part `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

type apiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}
