package classify_test

import (
	"context"
	"os"
	"testing"

	"expensetracker/internal/classify"
)

// TestLiveClassifyAgainstTheRealAPI is the only test here that leaves the
// machine. Everything else in this package answers from an httptest server,
// which proves the code agrees with the API as documented -- not that the
// documentation is right. This is the test that closes that gap, and it is
// the reason to run it before trusting a model or a key in production.
//
// It skips unless GEMINI_API_KEY is set, the same way the DB-touching tests
// skip without TEST_DATABASE_URL, so `go test ./...` stays green (and free,
// and offline) on a machine that has no key. Set GEMINI_MODEL too to check
// a different model before putting it in Render:
//
//	GEMINI_API_KEY=... go test ./internal/classify/ -run TestLive -v
//
// It asserts only that the answer is one of the candidates offered, never
// which one: the point is that the request shape, the schema and the
// response parsing all survive a real round trip. Which category a model
// picks is quality, not correctness, so the choice is logged for a human to
// judge rather than pinned to a value that would make this test flaky.
func TestLiveClassifyAgainstTheRealAPI(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set; skipping the live API check")
	}

	c := classify.New(classify.Config{APIKey: apiKey, Model: os.Getenv("GEMINI_MODEL")})
	if !c.Configured() {
		t.Fatal("Configured() = false with GEMINI_API_KEY set, want true")
	}

	// A real MB transfer note, in the shape bankmail actually extracts:
	// no diacritics, a merchant name buried in bank boilerplate.
	const note = "CHUYEN TIEN GRAB THANH TOAN CHUYEN DI"

	byID := make(map[int64]string, len(testCategories))
	for _, cat := range testCategories {
		byID[cat.ID] = cat.Name
	}

	got, err := c.Classify(context.Background(), note, testCategories)
	if err != nil {
		t.Fatalf("Classify(%q) error = %v, want nil -- the request shape or the model name does not match the live API", note, err)
	}
	name, ok := byID[got]
	if !ok {
		t.Fatalf("Classify(%q) = %d, which is not one of the candidates offered", note, got)
	}
	t.Logf("live API answered %d (%s) for %q", got, name, note)
}
