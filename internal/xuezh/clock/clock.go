package clock

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"time"
)

var durationRe = regexp.MustCompile(`^(?P<n>[0-9]+)(?P<unit>[dh])$`)

func NowUTC() (time.Time, error) {
	if override := os.Getenv("XUEZH_TEST_NOW_ISO"); override != "" {
		return ParseUTCISO(override)
	}
	return time.Now().UTC(), nil
}

func ParseUTCISO(s string) (time.Time, error) {
	if len(s) >= 1 && s[len(s)-1] == 'Z' {
		s = s[:len(s)-1] + "+00:00"
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		if t.Location() == time.UTC {
			return t.UTC(), nil
		}
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("timestamp must be timezone-aware (include +00:00 or Z)")
}

func ParseDuration(s string) (time.Duration, error) {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, errors.New("invalid duration: expected Nd or Nh, e.g. 30d, 24h")
	}
	var nStr, unit string
	for i, name := range durationRe.SubexpNames() {
		if name == "n" {
			nStr = m[i]
		}
		if name == "unit" {
			unit = m[i]
		}
	}
	n, err := strconv.Atoi(nStr)
	if err != nil {
		return 0, errors.New("invalid duration: expected Nd or Nh, e.g. 30d, 24h")
	}
	switch unit {
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	case "h":
		return time.Duration(n) * time.Hour, nil
	}
	return 0, errors.New("invalid duration: expected Nd or Nh, e.g. 30d, 24h")
}

func FormatISO(t time.Time) string {
	t = t.UTC().Truncate(time.Microsecond)
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05-07:00")
	}
	return t.Format("2006-01-02T15:04:05.000000-07:00")
}
