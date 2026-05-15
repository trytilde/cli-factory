package discovery

import (
	"context"
	"testing"

	"cli-factory/internal/app"
)

func TestSearchExcludesScoresAtOrBelowThreshold(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	results, err := Search(context.Background(), registry, nil, "gmail")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %#v, want none because gmail matches score exactly 0.8", results)
	}
}

func TestSearchIncludesOnlyScoresAboveThreshold(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	results, err := Search(context.Background(), registry, nil, "gmail send")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	for _, result := range results {
		if result.Score <= minimumSearchScore {
			t.Fatalf("result score = %v, want > %v: %#v", result.Score, minimumSearchScore, result)
		}
	}
}

func TestSearchLimitsResultsToTopFive(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	results, err := Search(context.Background(), registry, increasingEmbedder{}, "google")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != maxSearchResults {
		t.Fatalf("results = %d, want %d", len(results), maxSearchResults)
	}
	for i := 1; i < len(results); i++ {
		if results[i-1].Score < results[i].Score {
			t.Fatalf("results not sorted by score: %#v", results)
		}
	}
}

type increasingEmbedder struct{}

func (increasingEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, nil
}
