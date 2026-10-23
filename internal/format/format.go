package format

import (
	"fmt"
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
