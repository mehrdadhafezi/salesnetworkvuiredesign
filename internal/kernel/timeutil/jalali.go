package timeutil

import (
	"errors"
	"fmt"
	"time"
)

// Jalali support is a calendar formatter/parser for presentation and evidence
// only (E0 §9E). It is never a storage type: callers persist Instant/SourceTime.
// The algorithm is the arithmetic Jalaali calendar based on the 33-year
// break table (valid for Jalali years -61 .. 3177).

// ErrInvalidJalaliDate reports an out-of-range or malformed Jalali date.
var ErrInvalidJalaliDate = errors.New("timeutil: invalid Jalali date")

// JalaliDate is a calendar date in the Jalali (Solar Hijri) calendar.
type JalaliDate struct {
	Year  int
	Month int // 1..12
	Day   int // 1..31
}

// String renders YYYY/MM/DD.
func (d JalaliDate) String() string { return fmt.Sprintf("%04d/%02d/%02d", d.Year, d.Month, d.Day) }

var jalaliBreaks = [...]int{-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178}

type jalCalResult struct {
	leap  int
	gy    int
	march int
}

func jalCal(jy int) (jalCalResult, error) {
	bl := len(jalaliBreaks)
	gy := jy + 621
	leapJ := -14
	jp := jalaliBreaks[0]
	if jy < jp || jy >= jalaliBreaks[bl-1] {
		return jalCalResult{}, fmt.Errorf("%w: year %d out of range", ErrInvalidJalaliDate, jy)
	}
	jump := 0
	for i := 1; i < bl; i++ {
		jm := jalaliBreaks[i]
		jump = jm - jp
		if jy < jm {
			break
		}
		leapJ += jump/33*8 + (jump%33)/4
		jp = jm
	}
	n := jy - jp
	leapJ += n/33*8 + (n%33+3)/4
	if jump%33 == 4 && jump-n == 4 {
		leapJ++
	}
	leapG := gy/4 - (gy/100+1)*3/4 - 150
	march := 20 + leapJ - leapG
	if jump-n < 6 {
		n = n - jump + (jump+4)/33*33
	}
	leap := ((n+1)%33 - 1) % 4
	if leap == -1 {
		leap = 4
	}
	return jalCalResult{leap: leap, gy: gy, march: march}, nil
}

// g2d converts a Gregorian date to a Julian day number.
func g2d(gy, gm, gd int) int {
	d := (gy+(gm-8)/6+100100)*1461/4 + (153*((gm+9)%12)+2)/5 + gd - 34840408
	return d - (gy+100100+(gm-8)/6)/100*3/4 + 752
}

// d2g converts a Julian day number to a Gregorian date.
func d2g(jdn int) (int, int, int) {
	j := 4*jdn + 139361631
	j += (4*jdn+183187720)/146097*3/4*4 - 3908
	i := (j%1461)/4*5 + 308
	gd := (i%153)/5 + 1
	gm := (i/153)%12 + 1
	gy := j/1461 - 100100 + (8-gm)/6
	return gy, gm, gd
}

// IsJalaliLeapYear reports whether jy has 30 days in Esfand.
func IsJalaliLeapYear(jy int) (bool, error) {
	r, err := jalCal(jy)
	if err != nil {
		return false, err
	}
	return r.leap == 0, nil
}

// JalaliMonthLength returns the number of days in month jm of year jy.
func JalaliMonthLength(jy, jm int) (int, error) {
	if jm < 1 || jm > 12 {
		return 0, fmt.Errorf("%w: month %d", ErrInvalidJalaliDate, jm)
	}
	switch {
	case jm <= 6:
		return 31, nil
	case jm <= 11:
		return 30, nil
	}
	leap, err := IsJalaliLeapYear(jy)
	if err != nil {
		return 0, err
	}
	if leap {
		return 30, nil
	}
	return 29, nil
}

// Validate checks that d is a real Jalali calendar date.
func (d JalaliDate) Validate() error {
	length, err := JalaliMonthLength(d.Year, d.Month)
	if err != nil {
		return err
	}
	if d.Day < 1 || d.Day > length {
		return fmt.Errorf("%w: %s", ErrInvalidJalaliDate, d)
	}
	return nil
}

func jalaliToJDN(d JalaliDate) (int, error) {
	if err := d.Validate(); err != nil {
		return 0, err
	}
	r, err := jalCal(d.Year)
	if err != nil {
		return 0, err
	}
	return g2d(r.gy, 3, r.march) + (d.Month-1)*31 - d.Month/7*(d.Month-7) + d.Day - 1, nil
}

func jdnToJalali(jdn int) (JalaliDate, error) {
	gy, _, _ := d2g(jdn)
	jy := gy - 621
	r, err := jalCal(jy)
	if err != nil {
		return JalaliDate{}, err
	}
	k := jdn - g2d(gy, 3, r.march)
	if k >= 0 {
		if k <= 185 {
			return JalaliDate{Year: jy, Month: 1 + k/31, Day: k%31 + 1}, nil
		}
		k -= 186
	} else {
		jy--
		k += 179
		if r.leap == 1 {
			k++
		}
	}
	return JalaliDate{Year: jy, Month: 7 + k/30, Day: k%30 + 1}, nil
}

// ToJalali converts the calendar date of t (in t's own location) to Jalali.
// Callers choose the presentation zone explicitly before converting.
func ToJalali(t time.Time) (JalaliDate, error) {
	return jdnToJalali(g2d(t.Year(), int(t.Month()), t.Day()))
}

// GregorianDate is a proleptic Gregorian calendar date.
type GregorianDate struct {
	Year  int
	Month time.Month
	Day   int
}

// FromJalali converts a Jalali date to its Gregorian calendar date.
func FromJalali(d JalaliDate) (GregorianDate, error) {
	jdn, err := jalaliToJDN(d)
	if err != nil {
		return GregorianDate{}, err
	}
	gy, gm, gd := d2g(jdn)
	return GregorianDate{Year: gy, Month: time.Month(gm), Day: gd}, nil
}
