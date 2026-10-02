package leavepolicy

import (
	"testing"
	"time"
)

func d(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }

func TestAccrued(t *testing.T) {
	cases := []struct {
		name string
		p    Policy
		join string
		as   string
		want int
	}{
		{"upfront", Policy{Accrual: Annual}, "2020-01-01", "2026-01-02", 12},
		{"monthly march", Policy{Accrual: Monthly}, "2020-01-01", "2026-03-10", 3},
		{"monthly after year", Policy{Accrual: Monthly}, "2020-01-01", "2027-02-01", 12},
		{"monthly joiner", Policy{Accrual: Monthly}, "2026-07-15", "2026-12-01", 6},
		{"not yet eligible", Policy{Accrual: Annual, EligibilityMonths: 12}, "2026-02-01", "2026-12-31", 0},
		{"eligible mid-year upfront", Policy{Accrual: Annual, EligibilityMonths: 12}, "2025-06-01", "2026-06-01", 12},
		{"before eligibility date", Policy{Accrual: Annual, EligibilityMonths: 12}, "2025-06-01", "2026-05-31", 0},
		{"eligible mid-year monthly", Policy{Accrual: Monthly, EligibilityMonths: 12}, "2025-06-01", "2026-12-01", 7},
	}
	for _, c := range cases {
		if got := Accrued(c.p, 12, d(c.join), 2026, d(c.as)); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestCarry(t *testing.T) {
	p := Policy{Accrual: Annual, CarryOverMax: 5}
	if got := Carry(p, 12, 4, d("2020-01-01"), 2026); got != 5 {
		t.Fatalf("capped carry = %d", got)
	}
	if got := Carry(p, 12, 10, d("2020-01-01"), 2026); got != 2 {
		t.Fatalf("partial carry = %d", got)
	}
	if got := Carry(p, 12, 14, d("2020-01-01"), 2026); got != 0 {
		t.Fatalf("overspent carry = %d", got)
	}
	if got := Carry(Policy{Accrual: Annual}, 12, 0, d("2020-01-01"), 2026); got != 0 {
		t.Fatalf("disabled carry = %d", got)
	}
}

func TestCarryExpiry(t *testing.T) {
	p := Policy{Accrual: Annual, CarryOverMax: 5, CarryOverExpiryMonths: 3}
	exp, ok := CarryExpiry(p, 2026)
	if !ok || exp.Format(time.DateOnly) != "2026-03-31" {
		t.Fatalf("expiry %v %v", exp, ok)
	}
	if UsableCarry(p, 5, 2, 2026, d("2026-03-31")) != 5 {
		t.Fatal("before expiry all carry is usable")
	}
	if UsableCarry(p, 5, 2, 2026, d("2026-04-01")) != 2 {
		t.Fatal("after expiry only carry spent by the expiry date counts")
	}
	if _, ok := CarryExpiry(Policy{CarryOverMax: 5}, 2026); ok {
		t.Fatal("no expiry configured")
	}
}
