package process

import (
	"errors"
	"strings"
	"testing"
)

func TestBudgetErrorsAreDistinguishable(t *testing.T) {
	// The two ceilings are unrelated and are fixed in different settings. A
	// caller must be able to tell them apart with errors.Is, because reporting a
	// full hourly throttle as "daily budget exceeded" sends an operator to stare
	// at a spend meter reading $0.00 while the real blocker goes unmentioned.
	if errors.Is(ErrHourlyCapReached, ErrBudgetExceeded) {
		t.Fatal("hourly cap must not match the budget error")
	}
	if errors.Is(ErrBudgetExceeded, ErrHourlyCapReached) {
		t.Fatal("budget error must not match the hourly cap")
	}
}

func TestBudgetErrorsNameTheirOwnSetting(t *testing.T) {
	// Each message must point at the knob that fixes it, not the other one.
	if got := ErrHourlyCapReached.Error(); !strings.Contains(got, "hourly") {
		t.Fatalf("hourly error should say so: %q", got)
	}
	if got := ErrBudgetExceeded.Error(); !strings.Contains(got, "budget") {
		t.Fatalf("budget error should say so: %q", got)
	}
	if strings.Contains(ErrHourlyCapReached.Error(), "budget") {
		t.Fatalf("hourly error must not mention budget: %q", ErrHourlyCapReached.Error())
	}
}
