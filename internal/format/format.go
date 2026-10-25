package format

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func commafy(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	l := len(s)
	if l <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	rem := l % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < l; i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// BTC writes an amount the way BIP 177 does: the ₿ sign and whole base
// units (what were called sats), so 1000 is "₿1,000" and -500 is "-₿500".
func BTC(amount int64) string {
	if amount < 0 {
		return "-₿" + commafy(-amount)
	}
	return "₿" + commafy(amount)
}

// Count writes a whole number with thousands separators: 1312 is "1,312".
func Count(n int64) string {
	return commafy(n)
}

// Seconds writes a duration given in seconds the way people read it:
// "2.4 s", "41 s", "7 min", "3 h".
func Seconds(s float64) string {
	switch {
	case s < 10:
		return strconv.FormatFloat(math.Round(s*10)/10, 'f', -1, 64) + " s"
	case s < 60:
		return fmt.Sprintf("%.0f s", s)
	case s < 3600:
		return fmt.Sprintf("%.0f min", math.Round(s/60))
	case s < 36000:
		return strconv.FormatFloat(math.Round(s/360)/10, 'f', -1, 64) + " h"
	default:
		return fmt.Sprintf("%.0f h", math.Round(s/3600))
	}
}

// Percent writes a percentage with at most two decimals: "99.72%", "100%".
func Percent(p float64) string {
	return strconv.FormatFloat(math.Round(p*100)/100, 'f', -1, 64) + "%"
}

// Stage writes a payment stage for people: "in_flight" is "In flight".
func Stage(s string) string {
	return Cap(strings.ReplaceAll(s, "_", " "))
}

// Cap capitalizes an API value (a status, a type) for display: "settled" → "Settled".
func Cap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func TimeAgo(t *time.Time) string {
	if t == nil {
		return "--"
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
