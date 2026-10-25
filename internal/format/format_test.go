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

func TestBTC(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{
		{"zero", 0, "₿0"},
		{"positive", 1000, "₿1,000"},
		{"large", 1234567, "₿1,234,567"},
		{"negative", -500, "-₿500"},
		{"negative thousands", -1234, "-₿1,234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BTC(tt.amount); got != tt.want {
				t.Errorf("BTC(%d) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestCap(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"settled", "Settled"},
		{"pending", "Pending"},
		{"Failed", "Failed"},
	}
	for _, tt := range tests {
		if got := Cap(tt.in); got != tt.want {
			t.Errorf("Cap(%q) = %q, want %q", tt.in, got, tt.want)
		}
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

func TestSeconds(t *testing.T) {
	for in, want := range map[float64]string{
		2.4: "2.4 s", 3: "3 s", 9.96: "10 s", 41: "41 s", 412: "7 min", 1260: "21 min",
		3600: "1 h", 5400: "1.5 h", 10800: "3 h", 86400: "24 h",
	} {
		if got := Seconds(in); got != want {
			t.Errorf("Seconds(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	for in, want := range map[float64]string{99.722: "99.72%", 100: "100%", 0: "0%", 99.5: "99.5%"} {
		if got := Percent(in); got != want {
			t.Errorf("Percent(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestCountAndStage(t *testing.T) {
	if got := Count(1312); got != "1,312" {
		t.Errorf("Count = %q", got)
	}
	for in, want := range map[string]string{"in_flight": "In flight", "on_hold": "On hold", "queued": "Queued", "": ""} {
		if got := Stage(in); got != want {
			t.Errorf("Stage(%q) = %q, want %q", in, got, want)
		}
	}
}
