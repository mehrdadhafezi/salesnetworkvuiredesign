package main

// R4 fixture: tests may compose the evaluators.
import (
	"testing"

	"example.test/mod/internal/custody/release"
	"example.test/mod/internal/finance/review"
)

func TestCompose(t *testing.T) {
	_ = release.Evaluate
	_ = review.EvaluateDecision
}
