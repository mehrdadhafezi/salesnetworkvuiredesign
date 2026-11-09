package main

// R4 fixture: production code wiring a decision-gated evaluator.
import (
	"example.test/mod/internal/finance/posting"
	"example.test/mod/internal/finance/review"
)

func main() {
	_ = posting.Apply
	_ = review.EvaluateDecision
}
