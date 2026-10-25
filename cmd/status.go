package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/config"
	"github.com/lnbotdev/cli/internal/format"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Service status, wallet status and API health",
	Long: `Show whether ln.bot is working: the overall status, each part (sending,
receiving, the node, webhooks) with its live numbers, any notice from the
operators, and the last 24 hours. Then, when a wallet is configured, its
details, balance and API connectivity.

The service part needs no API key or wallet. The command exits 0 whatever
state the service is in; scripts read --json. Only failing to reach the API
is an error.`,
	Example: `  lnbot status
  lnbot status --wallet wal_abc
  lnbot status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		service, history, err := fetchServiceStatus(ctx)
		if err != nil {
			return err
		}

		w, err := statusWallet()
		if err != nil {
			return err
		}

		var (
			wal       *lnbot.Wallet
			firstAddr string
			latency   time.Duration
		)
		if w != nil {
			t0 := time.Now()
			wal, err = w.Get(ctx)
			if err != nil {
				return apiError("fetching status", err)
			}
			latency = time.Since(t0)

			addrs, addrErr := w.Addresses.List(ctx)
			if addrErr == nil && len(addrs) > 0 {
				firstAddr = addrs[0].Address
			}
		}

		if jsonFlag {
			out := map[string]any{
				"service":    service,
				"summary24h": history.Summary,
			}
			if wal != nil {
				out["walletId"] = wal.WalletID
				out["name"] = wal.Name
				out["balance"] = wal.Balance
				out["available"] = wal.Available
				out["onHold"] = wal.OnHold
				out["address"] = firstAddr
				out["latencyMs"] = latency.Milliseconds()
			}
			return json.NewEncoder(os.Stdout).Encode(out)
		}

		printService(service, &history.Summary)
		if wal == nil {
			return nil
		}

		fmt.Println()
		fmt.Printf("  Name:      %s\n", wal.Name)
		fmt.Printf("  ID:        %s\n", wal.WalletID)
		if firstAddr != "" {
			fmt.Printf("  Address:   %s\n", firstAddr)
		}
		fmt.Printf("  Balance:   %s\n", format.BTC(wal.Balance))
		fmt.Printf("  Available: %s\n", format.BTC(wal.Available))
		fmt.Printf("  On hold:   %s\n", format.BTC(wal.OnHold))
		fmt.Printf("  API:       ✓ Connected (%dms)\n", latency.Milliseconds())
		return nil
	},
}

// fetchServiceStatus reads the service's status now and over the last 24
// hours. Both routes are public, so it sends no key.
func fetchServiceStatus(ctx context.Context) (*lnbot.ServiceStatus, *lnbot.StatusHistory, error) {
	ln := config.AnonClient()
	service, err := ln.Status(ctx)
	if err != nil {
		return nil, nil, serviceStatusError(err)
	}
	history, err := ln.StatusHistory(ctx, nil)
	if err != nil {
		return nil, nil, serviceStatusError(err)
	}
	return service, history, nil
}

func serviceStatusError(err error) error {
	var tooMany *lnbot.TooManyRequestsError
	if errors.As(err, &tooMany) {
		return fmt.Errorf("checking service status: too many requests (the status routes allow 10 a minute per IP); try again in a minute")
	}
	return apiError("checking service status", err)
}

// statusWallet returns the wallet to show, or nil when none is configured.
func statusWallet() (*lnbot.WalletHandle, error) {
	if cfg == nil || (walletFlag == "" && cfg.ActiveWalletID == "") {
		return nil, nil
	}
	return resolveWallet()
}

// knownStages orders the stages of payments in progress as a payment
// moves through them.
var knownStages = []string{
	lnbot.PaymentStageQueued,
	lnbot.PaymentStageRetrying,
	lnbot.PaymentStageSending,
	lnbot.PaymentStageInFlight,
	lnbot.PaymentStageConfirming,
	lnbot.PaymentStageOnHold,
}

func printService(s *lnbot.ServiceStatus, sum *lnbot.StatusSummary) {
	fmt.Printf("  ln.bot:    %s\n", overallState(s))
	if n := s.Notice; n != nil {
		fmt.Printf("  Notice:    %s, since %s: %s\n", format.Cap(n.Level), clock(n.Since), n.Message)
	}
	fmt.Println()

	c := s.Components
	row := func(name, state, detail string) {
		fmt.Printf("  %-10s %-12s %s\n", name, state, detail)
	}

	snd := c.Sending
	row("Sending", format.Cap(snd.Status), inProgress(&snd))
	last := fmt.Sprintf("Last hour: %s sent, %s failed (%s on ln.bot's side)",
		format.Count(int64(snd.SentLastHour)), format.Count(int64(snd.FailedLastHour)), format.Count(int64(snd.ServiceFailuresLastHour)))
	row("", "", last+settleTimes(snd.MedianSeconds, snd.P95Seconds))

	received := "Last hour: " + format.Count(int64(c.Receiving.CreditedLastHour)) + " credited"
	if b := c.Receiving.CreditBacklog; b != nil {
		if *b == 0 {
			received += ", none waiting"
		} else {
			received += ", " + format.Count(*b) + " waiting"
		}
	}
	row("Receiving", format.Cap(c.Receiving.Status), received)

	node := "Didn't answer"
	if ms := c.Node.LatencyMs; ms != nil {
		node = "Answered in " + format.Count(*ms) + " ms"
	}
	row("Node", format.Cap(c.Node.Status), node)

	hooks := "Backlog unknown"
	if b := c.Webhooks.Backlog; b != nil {
		switch *b {
		case 0:
			hooks = "None waiting"
		case 1:
			hooks = "1 delivery waiting"
		default:
			hooks = format.Count(*b) + " deliveries waiting"
		}
	}
	row("Webhooks", format.Cap(c.Webhooks.Status), hooks)

	fmt.Println()
	if sum.Samples == 0 {
		fmt.Println("  Last 24 h: not measured")
		return
	}
	uptime := "no measured uptime"
	if up := sum.Uptime.Overall.Operational; up != nil {
		uptime = format.Percent(*up) + " uptime"
	}
	fmt.Printf("  Last 24 h: %s, %s sent, %s failed (%s on ln.bot's side)%s\n",
		uptime, format.Count(int64(sum.Sent)), format.Count(int64(sum.Failed)), format.Count(int64(sum.ServiceFailures)),
		settleTimes(sum.MedianSeconds, sum.P95Seconds))
}

func overallState(s *lnbot.ServiceStatus) string {
	switch s.Status {
	case lnbot.ServiceUnknown:
		if s.UpdatedAt == nil {
			return "⚠ Not measured yet"
		}
		return "⚠ Not measured since " + clock(*s.UpdatedAt)
	case lnbot.ServiceOperational:
		return "✓ Operational" + updated(s.UpdatedAt)
	default:
		return "⚠ " + format.Cap(s.Status) + updated(s.UpdatedAt)
	}
}

func updated(t *time.Time) string {
	if t == nil {
		return ""
	}
	return " (updated " + format.TimeAgo(t) + ")"
}

// inProgress describes the payments being sent: how many, at which stage,
// and how long the oldest has waited.
func inProgress(s *lnbot.SendingStatus) string {
	if s.InProgress == 0 {
		return "None in progress"
	}
	out := format.Count(int64(s.InProgress)) + " in progress"

	var parts []string
	seen := map[string]bool{}
	for _, stage := range knownStages {
		if n := s.ByStage[stage]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %s", format.Count(int64(n)), strings.ReplaceAll(stage, "_", " ")))
		}
		seen[stage] = true
	}
	var other []string
	for stage, n := range s.ByStage {
		if !seen[stage] && n > 0 {
			other = append(other, fmt.Sprintf("%s %s", format.Count(int64(n)), strings.ReplaceAll(stage, "_", " ")))
		}
	}
	sort.Strings(other)
	if parts = append(parts, other...); len(parts) > 0 {
		out += " (" + strings.Join(parts, ", ") + ")"
	}

	if w := s.LongestWaitSeconds; w != nil {
		out += ", oldest " + format.Seconds(*w)
	}
	return out
}

func settleTimes(median, p95 *float64) string {
	if median == nil || p95 == nil {
		return ""
	}
	return ", median " + format.Seconds(*median) + ", p95 " + format.Seconds(*p95)
}

// clock writes a time of day in local time, with the date when it isn't today.
func clock(t time.Time) string {
	t = t.Local()
	if t.Format("2006-01-02") != time.Now().Format("2006-01-02") {
		return t.Format("Jan 2 15:04")
	}
	return t.Format("15:04")
}
