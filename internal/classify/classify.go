// Package classify asks Claude to pick a category for a bank-transfer note,
// choosing only among the categories the caller says an account is allowed
// to use.
//
// Shaped like internal/mailer on purpose: a Config, a New/NewWithEndpoint
// pair (the latter is what lets a test point the client at an httptest
// server instead of the real API), a Configured check, and a call that
// takes ctx first. Callers must treat every error this package returns --
// not configured, a network failure, a 429/500, a malformed or
// out-of-candidate answer -- as equivalent to "could not classify" and fall
// back accordingly. See internal/handlers/inbox_process.go's
// resolveCategoryForNotice for why: a classification failure is allowed to
// cost a wrong category, never a whole transaction.
package classify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// defaultModel is used when Config.Model is empty. internal/config.Load
// already applies this same default to ANTHROPIC_MODEL, so in practice
// Classify only falls back to it when a test builds a Config by hand.
const defaultModel = "claude-opus-5"

// classifyTimeout bounds a single Classify call. This is what keeps a
// hung or merely slow Anthropic request from stalling
// processPendingEmails's goroutine indefinitely -- the background loop has
// no other timeout of its own.
const classifyTimeout = 20 * time.Second

// maxTokens is generous for a one-line JSON answer (a category id) but
// still small enough to keep a runaway response cheap.
const maxTokens = 256

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

// Config holds the Anthropic API settings. Both fields are optional, the
// same way internal/mailer.Config treats the Brevo settings: an empty
// APIKey just means Classify returns ErrNotConfigured (logged by the
// caller, not fatal) rather than the app refusing to start.
type Config struct {
	APIKey string
	Model  string
}

// Classifier calls Claude to classify a note against a caller-supplied list
// of categories.
type Classifier struct {
	cfg    Config
	client anthropic.Client
}

// New constructs a Classifier that calls the real Anthropic API.
func New(cfg Config) *Classifier {
	return newClassifier(cfg, option.WithAPIKey(cfg.APIKey))
}

// NewWithEndpoint is New with the API base URL overridable, so tests can
// point Classify at an httptest server instead of the real Anthropic API.
func NewWithEndpoint(cfg Config, endpoint string) *Classifier {
	return newClassifier(cfg, option.WithAPIKey(cfg.APIKey), option.WithBaseURL(endpoint))
}

func newClassifier(cfg Config, opts ...option.RequestOption) *Classifier {
	return &Classifier{cfg: cfg, client: anthropic.NewClient(opts...)}
}

// Configured reports whether enough settings are present to attempt a call.
// Callers use this to skip the call (and log a clear "not configured"
// reason) instead of letting Classify fail with an opaque 401 against an
// empty API key.
func (c *Classifier) Configured() bool {
	return c.cfg.APIKey != ""
}

// classifyAnswer is the shape Claude's structured output is constrained to.
// A single required integer field, nothing else -- there is no prose to
// parse and no way for the model to answer with anything but a category id.
type classifyAnswer struct {
	CategoryID int64 `json:"category_id"`
}

// Classify asks which of cats best fits note and returns that category's
// ID. It never returns an ID that was not in cats: the JSON schema sent to
// the model constrains category_id to the exact set of candidate ids via
// an enum, and the answer is checked against cats again here as defense in
// depth against a bug in that constraint (or, in principle, the model
// ignoring it).
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

	model := c.cfg.Model
	if model == "" {
		model = defaultModel
	}

	ctx, cancel := context.WithTimeout(ctx, classifyTimeout)
	defer cancel()

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildPrompt(note, cats))),
		},
		OutputConfig: anthropic.OutputConfigParam{
			// "low": this is a one-line classification among a short list
			// of known-good options, the cheapest effort level that still
			// produces a real answer -- not the multi-step reasoning
			// effort exists for.
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchema(cats)},
		},
	})
	if err != nil {
		return 0, wrapAPIError(err)
	}

	var answer classifyAnswer
	if err := json.Unmarshal([]byte(responseText(message)), &answer); err != nil {
		return 0, fmt.Errorf("classify: parse response: %w", err)
	}

	for _, cand := range cats {
		if cand.ID == answer.CategoryID {
			return answer.CategoryID, nil
		}
	}
	return 0, fmt.Errorf("classify: model answered category %d, which is not one of the %d candidates offered", answer.CategoryID, len(cats))
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

// answerSchema builds the JSON schema passed as OutputConfig.Format.
// Constraining category_id with an "enum" of the exact candidate ids -- not
// just "type: integer" -- is what makes the structured output do double
// duty as validation: a well-behaved model literally cannot express an id
// outside cats in its response.
func answerSchema(cats []Category) map[string]any {
	ids := make([]any, len(cats))
	for i, cat := range cats {
		ids[i] = cat.ID
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category_id": map[string]any{
				"type":        "integer",
				"enum":        ids,
				"description": "id of the chosen category, one of the candidates given",
			},
		},
		"required":             []any{"category_id"},
		"additionalProperties": false,
	}
}

// responseText concatenates every text block in message. The SDK also
// synthesises thinking/other block types, none of which carry the
// structured-output JSON, so only "text" blocks are collected.
func responseText(message *anthropic.Message) string {
	var b strings.Builder
	for _, block := range message.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

// wrapAPIError adds the HTTP status code to the error message when err came
// from the Anthropic API, which is what makes a 429 read differently from a
// 500 in a log line without the caller string-matching anything. Every
// status still becomes a plain error a caller treats identically (fall
// back); the switch exists for a clearer message, not different handling.
func wrapAPIError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429:
			return fmt.Errorf("classify: rate limited (429): %w", err)
		default:
			return fmt.Errorf("classify: anthropic api error (%d): %w", apiErr.StatusCode, err)
		}
	}
	return fmt.Errorf("classify: %w", err)
}
