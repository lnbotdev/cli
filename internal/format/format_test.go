package format

import (
	"testing"
	"time"
)

func TestCommafy(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, "0"},
		{"single digit", 5, "5"},
		{"two digits", 42, "42"},
		{"three digits", 999, "999"},
		{"four digits", 1000, "1,000"},
		{"thousands", 1234, "1,234"},
		{"millions", 1234567, "1,234,567"},
		{"large number", 1000000000, "1,000,000,000"},
		{"negative small", -5, "-5"},
		{"negative thousands", -1234, "-1,234"},
		{"negative millions", -1234567, "-1,234,567"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commafy(tt.n); got != tt.want {
				t.Errorf("commafy(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestSats(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{
		{"zero", 0, "0 sats"},
		{"positive", 1000, "1,000 sats"},
		{"large", 1234567, "1,234,567 sats"},
		{"negative", -500, "-500 sats"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sats(tt.amount); got != tt.want {
				t.Errorf("Sats(%d) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestSatsPlain(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{
		{"zero", 0, "0"},
		{"positive", 1000, "1,000"},
		{"negative", -500, "-500"},
		{"large", 1234567, "1,234,567"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SatsPlain(tt.amount); got != tt.want {
				t.Errorf("SatsPlain(%d) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		t    *time.Time
		want string
	}{
		{"nil", nil, "--"},
		{"just now", ptr(now.Add(-10 * time.Second)), "just now"},
		{"30 seconds ago", ptr(now.Add(-30 * time.Second)), "just now"},
		{"1 minute ago", ptr(now.Add(-90 * time.Second)), "1m ago"},
		{"5 minutes ago", ptr(now.Add(-5 * time.Minute)), "5m ago"},
		{"59 minutes ago", ptr(now.Add(-59 * time.Minute)), "59m ago"},
		{"1 hour ago", ptr(now.Add(-90 * time.Minute)), "1h ago"},
		{"5 hours ago", ptr(now.Add(-5 * time.Hour)), "5h ago"},
		{"23 hours ago", ptr(now.Add(-23 * time.Hour)), "23h ago"},
		{"1 day ago", ptr(now.Add(-36 * time.Hour)), "1d ago"},
		{"3 days ago", ptr(now.Add(-72 * time.Hour)), "3d ago"},
		{"30 days ago", ptr(now.Add(-30 * 24 * time.Hour)), "30d ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TimeAgo(tt.t); got != tt.want {
				t.Errorf("TimeAgo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"within limit", "hello", 10, "hello"},
		{"exact limit", "hello", 5, "hello"},
		{"over limit", "hello world", 8, "hello..."},
		{"max 3", "hello", 3, "hel"},
		{"max 2", "hello", 2, "he"},
		{"max 1", "hello", 1, "h"},
		{"max 4", "hello world", 4, "h..."},
		{"empty string", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.s, tt.max); got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func ptr(t time.Time) *time.Time { return &t }
