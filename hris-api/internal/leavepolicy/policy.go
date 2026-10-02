// Package leavepolicy computes annual-leave entitlement: upfront or monthly
// accrual, an optional service period before leave is earned, and capped
// carry-over of the previous year's unused entitlement.
package leavepolicy

import "time"

const (
	Annual  = "annual"  // full yearly allowance from the start of the year
	Monthly = "monthly" // allowance/12 per month, earned through the leave month
)

type Policy struct {
	Accrual           string
	CarryOverMax      int // days; 0 disables carry-over
	EligibilityMonths int // service months before annual leave is earned
	// CarryOverExpiryMonths limits carried-over days to leave taken up to the
	// end of this month of the year (0 = no expiry).
	CarryOverExpiryMonths int
}

// CarryExpiry is the last date carried-over days can be used in a year, and
// false when carried days never expire.
func CarryExpiry(p Policy, year int) (time.Time, bool) {
	if p.CarryOverExpiryMonths <= 0 || p.CarryOverMax <= 0 {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(p.CarryOverExpiryMonths)+1, 0, 0, 0, 0, 0, time.UTC), true
}

// UsableCarry is how much of the carried-over days still counts at asOf:
// all of it before expiry, afterwards only what was spent by the expiry date.
func UsableCarry(p Policy, carry, spentByExpiry, year int, asOf time.Time) int {
	expiry, ok := CarryExpiry(p, year)
	if !ok || !asOf.After(expiry) {
		return carry
	}
	return min(carry, spentByExpiry)
}

// Accrued is the entitlement of a year earned by asOf. Months before the
// employee became eligible earn nothing; with monthly accrual the month of
// asOf counts as earned.
func Accrued(p Policy, allowance int, joined time.Time, year int, asOf time.Time) int {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	eligible := joined.AddDate(0, p.EligibilityMonths, 0)
	if eligible.After(end) || asOf.Before(start) || asOf.Before(eligible) {
		return 0
	}
	if p.Accrual != Monthly {
		return allowance
	}
	first := 1
	if eligible.After(start) {
		first = int(eligible.Month())
	}
	last := 12
	if !asOf.After(end) {
		last = int(asOf.Month())
	}
	return allowance * (last - first + 1) / 12
}

// Carry is the unused entitlement of the previous year brought forward,
// capped at CarryOverMax. Only the previous year's own entitlement carries;
// days carried into it do not roll again.
func Carry(p Policy, prevAllowance, prevUsed int, joined time.Time, year int) int {
	if p.CarryOverMax <= 0 {
		return 0
	}
	prevEnd := time.Date(year-1, time.December, 31, 0, 0, 0, 0, time.UTC)
	unused := Accrued(p, prevAllowance, joined, year-1, prevEnd) - prevUsed
	return max(0, min(unused, p.CarryOverMax))
}
