package payroll

import (
	"testing"
	"time"
)

func TestTERRateBoundaries(t *testing.T) {
	cases := []struct {
		status string
		gross  int64
		rate   int64
	}{
		{"TK/0", 5_400_000, 0},
		{"TK/0", 5_400_001, 25},
		{"K/0", 10_000_000, 200},
		{"TK/1", 1_400_000_001, 3400},
		{"K/1", 10_000_000, 150},
		{"TK/2", 9_200_000, 100},
		{"K/2", 9_200_001, 150},
		{"K/3", 6_600_000, 0},
		{"K/3", 12_950_000, 300},
	}
	for _, c := range cases {
		if got := TERRate(c.status, c.gross); got != c.rate {
			t.Errorf("TERRate(%s, %d) = %d, want %d", c.status, c.gross, got, c.rate)
		}
	}
	if TERTax("TK/0", 10_000_000) != 200_000 {
		t.Fatal("TK/0 Rp10jt should withhold 2%")
	}
}

func TestPasal17(t *testing.T) {
	for _, c := range []struct{ pkp, tax int64 }{
		{0, 0},
		{60_000_000, 3_000_000},
		{300_000_000, 44_000_000},
		// 3jt + 28.5jt + 62.5jt + 1.35M + 175jt
		{5_500_000_000, 1_619_000_000},
	} {
		if got := Pasal17(c.pkp); got != c.tax {
			t.Errorf("Pasal17(%d) = %d, want %d", c.pkp, got, c.tax)
		}
	}
}

func TestAnnualTaxTrueUp(t *testing.T) {
	// TK/0 earning Rp10jt every month: Jan–Nov withhold 2% (Rp200k).
	ytd := YearToDate{Gross: 110_000_000, Tax: 11 * 200_000, Months: 11}
	// Annual: 120jt − 6jt biaya jabatan − 54jt PTKP = 60jt PKP → 3jt.
	if got := AnnualTax("TK/0", ytd, 10_000_000, 0); got != 800_000 {
		t.Fatalf("December tax = %d, want 800000", got)
	}
	// Over-withholding becomes a refund.
	ytd.Tax = 4_000_000
	if got := AnnualTax("TK/0", ytd, 10_000_000, 0); got != -1_000_000 {
		t.Fatalf("refund = %d, want -1000000", got)
	}
}

func TestComputeGross(t *testing.T) {
	r := Compute(Input{
		Lines: []Line{
			{Kind: Earning, Code: CodeBasic, Name: "Gaji pokok", Amount: 9_000_000, Taxable: true, Fixed: true},
			{Kind: Earning, Code: CodeAllowance, Name: "Transport", Amount: 1_000_000, Taxable: true},
		},
		PTKP: "TK/0", TaxMethod: TaxGross,
		Enrollment: Enrollment{Kesehatan: true, Ketenagakerjaan: true, Pensiun: true},
		Settings:   DefaultSettings,
	})
	want := map[string]int64{
		CodeKesEmployer: 360_000, CodeKesEmployee: 90_000,
		CodeJHTEmployer: 333_000, CodeJHTEmployee: 180_000,
		CodeJPEmployer: 180_000, CodeJPEmployee: 90_000,
		CodeJKK: 21_600, CodeJKM: 27_000,
	}
	got := map[string]int64{}
	for _, l := range r.Lines {
		got[l.Code] = l.Amount
	}
	for code, amount := range want {
		if got[code] != amount {
			t.Errorf("%s = %d, want %d", code, got[code], amount)
		}
	}
	// Gross for tax: 10jt + Kes ER 360k + JKK 21.6k + JKM 27k = 10,408,600 → 2.5%.
	if r.TaxableGross != 10_408_600 || r.Tax != 260_215 || got[CodePPh21] != 260_215 {
		t.Fatalf("taxable %d tax %d", r.TaxableGross, r.Tax)
	}
	if r.Pension != 270_000 || r.Net != 10_000_000-90_000-180_000-90_000-260_215 {
		t.Fatalf("pension %d net %d", r.Pension, r.Net)
	}
	if r.EmployerCost != 360_000+333_000+180_000+21_600+27_000 {
		t.Fatalf("employer cost %d", r.EmployerCost)
	}
}

func TestComputeCapsAndGrossUp(t *testing.T) {
	r := Compute(Input{
		Lines: []Line{{Kind: Earning, Code: CodeBasic, Amount: 30_000_000, Taxable: true, Fixed: true}},
		PTKP:  "K/1", TaxMethod: TaxGrossUp,
		Enrollment: Enrollment{Kesehatan: true, Pensiun: true},
		Settings:   DefaultSettings,
	})
	var kes, jp, allowance int64
	for _, l := range r.Lines {
		switch l.Code {
		case CodeKesEmployee:
			kes = l.Amount
		case CodeJPEmployee:
			jp = l.Amount
		case CodeTaxAllowance:
			allowance = l.Amount
		}
	}
	if kes != 120_000 || jp != 105_474 {
		t.Fatalf("caps not applied: kes %d jp %d", kes, jp)
	}
	if allowance == 0 || allowance != r.Tax {
		t.Fatalf("gross-up allowance %d must equal tax %d", allowance, r.Tax)
	}
	if r.Tax != TERTax("K/1", r.TaxableGross) {
		t.Fatalf("allowance is not a fixed point")
	}
}

func TestComputeRefundAndNone(t *testing.T) {
	base := []Line{{Kind: Earning, Code: CodeBasic, Amount: 10_000_000, Taxable: true, Fixed: true}}
	r := Compute(Input{Lines: base, PTKP: "TK/0", TaxMethod: TaxGross, Final: true,
		YTD: YearToDate{Gross: 110_000_000, Tax: 4_000_000, Months: 11}})
	if r.Tax != -1_000_000 || r.Net != 11_000_000 {
		t.Fatalf("refund tax %d net %d", r.Tax, r.Net)
	}
	r = Compute(Input{Lines: base, PTKP: "TK/0", TaxMethod: TaxNone})
	if r.Tax != 0 || r.TaxableGross != 0 || r.Net != 10_000_000 || len(r.Lines) != 1 {
		t.Fatalf("tax none must leave the slip unchanged: %+v", r)
	}
}

func TestOvertimePay(t *testing.T) {
	wage := int64(1_730_000) // Rp10,000 per hour
	cases := []struct {
		minutes       int
		rest, sixDays bool
		want          int64
	}{
		{60, false, false, 15_000},
		{180, false, false, 15_000 + 40_000},
		{30, false, false, 7_500},
		{9 * 60, true, false, 8*20_000 + 30_000},
		{11 * 60, true, false, 8*20_000 + 30_000 + 2*40_000},
		{8 * 60, true, true, 7*20_000 + 30_000},
	}
	for _, c := range cases {
		if got := OvertimePay(wage, c.minutes, c.rest, c.sixDays); got != c.want {
			t.Errorf("OvertimePay(%d min, rest=%v, six=%v) = %d, want %d", c.minutes, c.rest, c.sixDays, got, c.want)
		}
	}
}

func TestTHRAndProrate(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }
	if THR(12_000_000, d("2025-01-15"), d("2026-03-20")) != 12_000_000 {
		t.Fatal("12+ months gets one month's wage")
	}
	if THR(12_000_000, d("2025-10-15"), d("2026-03-20")) != 5_000_000 {
		t.Fatal("5 months gets 5/12")
	}
	if THR(12_000_000, d("2026-03-01"), d("2026-03-20")) != 0 {
		t.Fatal("under one month gets nothing")
	}
	if ServiceMonths(d("2025-01-31"), d("2025-02-28")) != 0 || ServiceMonths(d("2025-01-31"), d("2025-03-31")) != 2 {
		t.Fatal("service months must count whole months")
	}
	cal := NewCalendar([]int64{1, 2, 3, 4, 5}, []string{"2026-03-20"})
	// March 2026: 22 weekdays, one holiday.
	if n := cal.Workdays(d("2026-03-01"), d("2026-03-31")); n != 21 {
		t.Fatalf("workdays = %d", n)
	}
	if Prorate(10_500_000, 7, 21) != 3_500_000 || Prorate(100, 21, 21) != 100 || Prorate(100, 0, 21) != 0 {
		t.Fatal("prorate")
	}
}

func TestComputeCorrectionDifference(t *testing.T) {
	// The original slip taxed Rp10jt at TER A 2% (200k). A Rp2jt bonus moves
	// the month to Rp12jt (4%, 480k): the correction withholds 280k more.
	r := Compute(Input{
		Lines:     []Line{{Kind: Earning, Code: CodeAdjustment, Name: "Bonus", Amount: 2_000_000, Taxable: true}},
		PTKP:      "TK/0",
		TaxMethod: TaxGross,
		Base:      YearToDate{Gross: 10_000_000, Tax: 200_000},
	})
	if r.Tax != 280_000 || r.Net != 1_720_000 {
		t.Fatalf("correction tax %d net %d", r.Tax, r.Net)
	}
	// An empty correction changes nothing.
	if r := Compute(Input{PTKP: "TK/0", TaxMethod: TaxGross, Base: YearToDate{Gross: 10_000_000, Tax: 200_000}}); r.Tax != 0 || len(r.Lines) != 0 {
		t.Fatalf("empty correction: %+v", r)
	}
}
