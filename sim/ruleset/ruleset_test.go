package ruleset

import (
	"strings"
	"testing"
)

func TestGetUnknownRulesetListsAvailable(t *testing.T) {
	Register(&Ruleset{ID: "test-get-unknown", RegisterSpecs: func() {}})
	_, err := Get("does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "test-get-unknown") {
		t.Fatalf("expected error listing available rulesets, got %v", err)
	}
}

func TestRegisterTwicePanics(t *testing.T) {
	Register(&Ruleset{ID: "test-dup", RegisterSpecs: func() {}})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	Register(&Ruleset{ID: "test-dup", RegisterSpecs: func() {}})
}

func TestActivateIsIdempotentAndExclusive(t *testing.T) {
	calls := 0
	Register(&Ruleset{ID: "test-a", RegisterSpecs: func() { calls++ }})
	Register(&Ruleset{ID: "test-b", RegisterSpecs: func() {}})

	if err := Activate("test-a"); err != nil {
		t.Fatal(err)
	}
	if err := Activate("test-a"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("RegisterSpecs called %d times, want 1", calls)
	}
	if Active() != "test-a" {
		t.Fatalf("Active() = %q, want test-a", Active())
	}
	if err := Activate("test-b"); err == nil {
		t.Fatal("expected error switching rulesets in one process")
	}
}
