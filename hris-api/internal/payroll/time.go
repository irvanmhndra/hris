package payroll

import "time"

// Calendar is the company's working week plus holidays (YYYY-MM-DD).
type Calendar struct {
	Week     map[time.Weekday]bool
	Holidays map[string]bool
}

func NewCalendar(workdays []int64, holidays []string) Calendar {
	c := Calendar{Week: map[time.Weekday]bool{}, Holidays: map[string]bool{}}
	for _, d := range workdays {
		c.Week[time.Weekday(d)] = true
	}
	for _, h := range holidays {
		c.Holidays[h] = true
	}
	return c
}

// IsWorkday reports whether the date is a scheduled working day.
func (c Calendar) IsWorkday(d time.Time) bool {
	return c.Week[d.Weekday()] && !c.Holidays[d.Format(time.DateOnly)]
}

// Workdays counts working days in [from, to], both inclusive.
func (c Calendar) Workdays(from, to time.Time) int {
	n := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if c.IsWorkday(d) {
			n++
		}
	}
	return n
}

// Prorate scales a monthly amount by worked/total working days, rounding to
// the nearest rupiah. A full month (or an empty calendar) is not scaled.
func Prorate(amount int64, worked, total int) int64 {
	if total <= 0 || worked >= total {
		return amount
	}
	if worked <= 0 {
		return 0
	}
	return (amount*int64(worked) + int64(total)/2) / int64(total)
}

// HourlyDivisor is the statutory 1/173 of monthly wage per overtime hour.
const HourlyDivisor = 173

// OvertimePay applies PP 35/2021 multipliers to one overtime session.
// On a working day the first hour is paid 1.5× and later hours 2×. On a rest
// day or public holiday a 5-day week pays 2× for 8 hours, 3× for the 9th, and
// 4× after; a 6-day week uses 7 hours, the 8th, and after.
func OvertimePay(monthlyWage int64, minutes int, restDay, sixDayWeek bool) int64 {
	// Multipliers are doubled so 1.5× stays an integer.
	type band struct{ until, mult2 int }
	var bands []band
	switch {
	case !restDay:
		bands = []band{{60, 3}, {1 << 30, 4}}
	case sixDayWeek:
		bands = []band{{7 * 60, 4}, {8 * 60, 6}, {1 << 30, 8}}
	default:
		bands = []band{{8 * 60, 4}, {9 * 60, 6}, {1 << 30, 8}}
	}
	var units int64 // minute × doubled multiplier
	prev := 0
	for _, b := range bands {
		if minutes <= prev {
			break
		}
		units += int64(min(minutes, b.until)-prev) * int64(b.mult2)
		prev = b.until
	}
	den := int64(HourlyDivisor * 60 * 2)
	return (monthlyWage*units + den/2) / den
}

// ServiceMonths counts whole months of continuous service from joined to at.
func ServiceMonths(joined, at time.Time) int {
	if at.Before(joined) {
		return 0
	}
	m := (at.Year()-joined.Year())*12 + int(at.Month()-joined.Month())
	if at.Day() < joined.Day() {
		m--
	}
	return max(m, 0)
}

// THR is the religious holiday allowance: one month's wage after 12 months
// of service, proportional (months/12) from one month, nothing before that.
func THR(monthlyWage int64, joined, at time.Time) int64 {
	m := ServiceMonths(joined, at)
	switch {
	case m >= 12:
		return monthlyWage
	case m >= 1:
		return monthlyWage * int64(m) / 12
	}
	return 0
}
