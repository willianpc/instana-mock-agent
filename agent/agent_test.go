package agent_test

import (
	"testing"

	at "github.com/willianpc/instana-mock-agent/internal/test"
)

func TestSum(t *testing.T) {
	expected := 90

	actual := at.Sum(2, 88)

	if actual != expected {
		t.Fatalf("expected: %v but received: %v", expected, actual)
	}
}
