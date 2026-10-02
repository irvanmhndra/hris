// Package payroll holds the pure Indonesian payroll rules: PPh 21 (TER,
// PP 58/2023 and PMK 168/2023), BPJS contributions, overtime pay
// (PP 35/2021), THR (Permenaker 6/2016), and working-day prorata. It has no
// database or HTTP dependencies, so every rule is unit-testable.
package payroll

import "math"

// Rates are expressed in hundredths of a percent: 25 = 0.25%, 10000 = 100%.
const rateScale = 10000

// PTKPStatuses lists the supported tax statuses (single/married + dependants).
var PTKPStatuses = []string{"TK/0", "TK/1", "TK/2", "TK/3", "K/0", "K/1", "K/2", "K/3"}

// PTKP returns the annual non-taxable income for a status, and false when the
// status is unknown.
func PTKP(status string) (int64, bool) {
	const self, married, dependant = 54_000_000, 4_500_000, 4_500_000
	switch status {
	case "TK/0":
		return self, true
	case "TK/1":
		return self + dependant, true
	case "TK/2":
		return self + 2*dependant, true
	case "TK/3":
		return self + 3*dependant, true
	case "K/0":
		return self + married, true
	case "K/1":
		return self + married + dependant, true
	case "K/2":
		return self + married + 2*dependant, true
	case "K/3":
		return self + married + 3*dependant, true
	}
	return 0, false
}

// TERCategory maps a PTKP status onto the monthly TER table it uses.
func TERCategory(status string) string {
	switch status {
	case "TK/0", "TK/1", "K/0":
		return "A"
	case "TK/2", "TK/3", "K/1", "K/2":
		return "B"
	case "K/3":
		return "C"
	}
	return ""
}

type bracket struct {
	upTo int64 // inclusive upper bound of monthly gross income
	rate int64 // hundredths of a percent
}

// Monthly TER tables (Lampiran PP 58/2023).
var terTables = map[string][]bracket{
	"A": {
		{5_400_000, 0}, {5_650_000, 25}, {5_950_000, 50}, {6_300_000, 75}, {6_750_000, 100},
		{7_500_000, 125}, {8_550_000, 150}, {9_650_000, 175}, {10_050_000, 200}, {10_350_000, 225},
		{10_700_000, 250}, {11_050_000, 300}, {11_600_000, 350}, {12_500_000, 400}, {13_750_000, 500},
		{15_100_000, 600}, {16_950_000, 700}, {19_750_000, 800}, {24_150_000, 900}, {26_450_000, 1000},
		{28_000_000, 1100}, {30_050_000, 1200}, {32_400_000, 1300}, {35_400_000, 1400}, {39_100_000, 1500},
		{43_850_000, 1600}, {47_800_000, 1700}, {51_400_000, 1800}, {56_300_000, 1900}, {62_200_000, 2000},
		{68_600_000, 2100}, {77_500_000, 2200}, {89_000_000, 2300}, {103_000_000, 2400}, {125_000_000, 2500},
		{157_000_000, 2600}, {206_000_000, 2700}, {337_000_000, 2800}, {454_000_000, 2900}, {550_000_000, 3000},
		{695_000_000, 3100}, {910_000_000, 3200}, {1_400_000_000, 3300}, {math.MaxInt64, 3400},
	},
	"B": {
		{6_200_000, 0}, {6_500_000, 25}, {6_850_000, 50}, {7_300_000, 75}, {9_200_000, 100},
		{10_750_000, 150}, {11_250_000, 200}, {11_600_000, 250}, {12_600_000, 300}, {13_600_000, 400},
		{14_950_000, 500}, {16_400_000, 600}, {18_450_000, 700}, {21_850_000, 800}, {26_000_000, 900},
		{27_700_000, 1000}, {29_350_000, 1100}, {31_450_000, 1200}, {33_950_000, 1300}, {37_100_000, 1400},
		{41_100_000, 1500}, {45_800_000, 1600}, {49_500_000, 1700}, {53_800_000, 1800}, {58_500_000, 1900},
		{64_000_000, 2000}, {71_000_000, 2100}, {80_000_000, 2200}, {93_000_000, 2300}, {109_000_000, 2400},
		{129_000_000, 2500}, {163_000_000, 2600}, {211_000_000, 2700}, {374_000_000, 2800}, {459_000_000, 2900},
		{555_000_000, 3000}, {704_000_000, 3100}, {957_000_000, 3200}, {1_405_000_000, 3300}, {math.MaxInt64, 3400},
	},
	"C": {
		{6_600_000, 0}, {6_950_000, 25}, {7_350_000, 50}, {7_800_000, 75}, {8_850_000, 100},
		{9_800_000, 125}, {10_950_000, 150}, {11_200_000, 175}, {12_050_000, 200}, {12_950_000, 300},
		{14_150_000, 400}, {15_550_000, 500}, {17_050_000, 600}, {19_500_000, 700}, {22_700_000, 800},
		{26_600_000, 900}, {28_100_000, 1000}, {30_100_000, 1100}, {32_600_000, 1200}, {35_400_000, 1300},
		{38_900_000, 1400}, {43_000_000, 1500}, {47_400_000, 1600}, {51_200_000, 1700}, {55_800_000, 1800},
		{60_400_000, 1900}, {66_700_000, 2000}, {74_500_000, 2100}, {83_200_000, 2200}, {95_600_000, 2300},
		{110_000_000, 2400}, {134_000_000, 2500}, {169_000_000, 2600}, {221_000_000, 2700}, {390_000_000, 2800},
		{463_000_000, 2900}, {561_000_000, 3000}, {709_000_000, 3100}, {965_000_000, 3200}, {1_419_000_000, 3300},
		{math.MaxInt64, 3400},
	},
}

// TERRate returns the monthly effective rate (hundredths of a percent) for a
// PTKP status and monthly gross income.
func TERRate(status string, gross int64) int64 {
	for _, b := range terTables[TERCategory(status)] {
		if gross <= b.upTo {
			return b.rate
		}
	}
	return 0
}

// TERTax is the monthly PPh 21 withheld in January–November (or any month
// that is not the employee's last tax period of the year), rounded down.
func TERTax(status string, gross int64) int64 {
	if gross <= 0 {
		return 0
	}
	return gross * TERRate(status, gross) / rateScale
}

// Pasal17 applies the annual progressive rates (UU HPP) to taxable income.
func Pasal17(pkp int64) int64 {
	layers := []struct{ upTo, pct int64 }{
		{60_000_000, 5}, {250_000_000, 15}, {500_000_000, 25}, {5_000_000_000, 30}, {math.MaxInt64, 35},
	}
	var tax, prev int64
	for _, l := range layers {
		if pkp <= prev {
			break
		}
		top := min(pkp, l.upTo)
		tax += (top - prev) * l.pct / 100
		prev = l.upTo
	}
	return tax
}

// YearToDate sums the employee's earlier tax periods of the same year that
// are already locked (finalized or paid).
type YearToDate struct {
	Gross   int64 // taxable gross, including employer-paid benefit premiums
	Pension int64 // employee JHT + JP contributions
	Tax     int64 // PPh 21 already withheld (negative for refunds)
	Months  int
}

// AnnualTax computes PPh 21 for the last tax period of the year: the full
// year's tax under Pasal 17 minus what January–November already withheld.
// The result is negative when too much was withheld (a refund).
func AnnualTax(status string, ytd YearToDate, gross, pension int64) int64 {
	ptkp, _ := PTKP(status)
	annualGross := ytd.Gross + gross
	months := int64(ytd.Months + 1)
	// Biaya jabatan: 5% of gross, capped at Rp500,000 per month worked
	// (Rp6,000,000 for a full year).
	jobCost := min(annualGross*5/100, 500_000*months)
	neto := annualGross - jobCost - ytd.Pension - pension
	pkp := max((neto-ptkp)/1000*1000, 0)
	return Pasal17(pkp) - ytd.Tax
}
