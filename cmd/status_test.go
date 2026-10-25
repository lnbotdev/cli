package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// The contract's examples: a degraded service with a notice, and its 24 hours.
const (
	statusDegraded = `{"status":"degraded","updatedAt":"2026-10-25T14:33:53.06+00:00","notice":{"level":"warning","message":"Sends are slower than usual.","since":"2026-10-25T14:20:00+00:00","until":null},"components":{"sending":{"status":"degraded","inProgress":3,"byStage":{"in_flight":2,"retrying":1},"longestWaitSeconds":412,"sentLastHour":57,"failedLastHour":4,"serviceFailuresLastHour":0,"medianSeconds":2.4,"p95Seconds":9.1},"receiving":{"status":"operational","creditedLastHour":120,"creditBacklog":0},"node":{"status":"operational","latencyMs":310},"webhooks":{"status":"operational","backlog":2}}}`
	history24h     = `{"range":"24h","bucketSeconds":300,"from":"2026-10-24T14:35:00+00:00","to":"2026-10-25T14:35:12+00:00","summary":{"samples":1438,"uptime":{"overall":{"operational":99.722,"degraded":0.278,"outage":0},"sending":{"operational":99.722,"degraded":0.278,"outage":0},"receiving":{"operational":100,"degraded":0,"outage":0},"node":{"operational":100,"degraded":0,"outage":0},"webhooks":{"operational":100,"degraded":0,"outage":0}},"sent":1312,"failed":21,"serviceFailures":2,"credited":2875,"medianSeconds":2.3,"p95Seconds":8.6,"settleTimes":[{"upToSeconds":1,"count":40},{"upToSeconds":null,"count":0}],"avgLatencyMs":296,"maxLatencyMs":2140},"buckets":[],"notices":[]}`
)

// serve sends every request the SDK makes to mux.
func serve(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	srv := httptest.NewServer(mux)
	u, _ := url.Parse(srv.URL)
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = redirect{u}
	t.Cleanup(func() {
		http.DefaultClient.Transport = old
		srv.Close()
	})
}

// statusAPI serves the status routes (status code code, body status) and a
// wallet, and records the Authorization header the status routes got.
func statusAPI(t *testing.T, code int, status string) *[]string {
	t.Helper()
	var mu sync.Mutex
	auth := []string{}
	writeJSON := func(w http.ResponseWriter, code int, body string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		fmt.Fprint(w, body)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auth = append(auth, r.Header.Get("Authorization"))
		mu.Unlock()
		writeJSON(w, code, status)
	})
	mux.HandleFunc("/v1/status/history", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auth = append(auth, r.Header.Get("Authorization"))
		mu.Unlock()
		if r.URL.Query().Get("range") != "24h" {
			t.Errorf("history range = %q", r.URL.Query().Get("range"))
		}
		writeJSON(w, 200, history24h)
	})
	mux.HandleFunc("/v1/wallets/wal_main123", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"walletId":"wal_main123","name":"main","balance":77,"onHold":0,"available":77}`)
	})
	mux.HandleFunc("/v1/wallets/wal_main123/addresses", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `[{"address":"x4k7m2@ln.bot","generated":true,"cost":0}]`)
	})
	serve(t, mux)
	return &auth
}

func TestStatus_ServiceWithoutWallet(t *testing.T) {
	setupNoConfig(t)
	auth := statusAPI(t, 200, statusDegraded)

	stdout, _, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, want := range []string{
		"  ln.bot:    ⚠ Degraded (updated ",
		"  Notice:    Warning, since ",
		": Sends are slower than usual.",
		"  Sending    Degraded     3 in progress (1 retrying, 2 in flight), oldest 7 min",
		"Last hour: 57 sent, 4 failed (0 on ln.bot's side), median 2.4 s, p95 9.1 s",
		"  Receiving  Operational  Last hour: 120 credited, none waiting",
		"  Node       Operational  Answered in 310 ms",
		"  Webhooks   Operational  2 deliveries waiting",
		"  Last 24 h: 99.72% uptime, 1,312 sent, 21 failed (2 on ln.bot's side), median 2.3 s, p95 8.6 s",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Name:") {
		t.Errorf("printed a wallet section without a wallet:\n%s", stdout)
	}
	for _, a := range *auth {
		if a != "" {
			t.Errorf("status routes got Authorization %q, want none", a)
		}
	}
}

func TestStatus_WithWallet(t *testing.T) {
	setupConfig(t, testConfig())
	auth := statusAPI(t, 200, statusDegraded)

	stdout, _, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	service := strings.Index(stdout, "ln.bot:")
	wallet := strings.Index(stdout, "  Name:      main")
	if service < 0 || wallet < service || !strings.Contains(stdout, "  Address:   x4k7m2@ln.bot") || !strings.Contains(stdout, "  API:       ✓ Connected") {
		t.Errorf("want the service, then the wallet:\n%s", stdout)
	}
	for _, a := range *auth {
		if a != "" {
			t.Errorf("status routes got Authorization %q, want none", a)
		}
	}
}

func TestStatus_JSON(t *testing.T) {
	for _, withWallet := range []bool{false, true} {
		t.Run(fmt.Sprintf("wallet=%v", withWallet), func(t *testing.T) {
			if withWallet {
				setupConfig(t, testConfig())
			} else {
				setupNoConfig(t)
			}
			statusAPI(t, 200, statusDegraded)

			stdout, _, err := executeCmd("status", "--json")
			if err != nil {
				t.Fatalf("status --json: %v", err)
			}
			var out map[string]json.RawMessage
			if err := json.Unmarshal([]byte(stdout), &out); err != nil {
				t.Fatalf("decoding %s: %v", stdout, err)
			}
			var service struct {
				Status     string
				Components struct {
					Sending struct{ ByStage map[string]int }
				}
			}
			json.Unmarshal(out["service"], &service)
			if service.Status != "degraded" || service.Components.Sending.ByStage["in_flight"] != 2 {
				t.Errorf("service = %s", out["service"])
			}
			var summary struct{ Sent int }
			json.Unmarshal(out["summary24h"], &summary)
			if summary.Sent != 1312 {
				t.Errorf("summary24h = %s", out["summary24h"])
			}
			if _, ok := out["walletId"]; ok != withWallet {
				t.Errorf("walletId present = %v, want %v", ok, withWallet)
			}
		})
	}
}

func TestStatus_UnknownStates(t *testing.T) {
	setupNoConfig(t)
	statusAPI(t, 200, strings.Replace(statusDegraded, `"status":"degraded","updatedAt"`, `"status":"maintenance","updatedAt"`, 1))
	stdout, _, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(stdout, "  ln.bot:    ⚠ Maintenance (updated ") {
		t.Errorf("unknown overall status:\n%s", stdout)
	}

	setupNoConfig(t)
	never := `{"status":"unknown","updatedAt":null,"notice":null,"components":{"sending":{"status":"unknown","inProgress":0,"byStage":{},"longestWaitSeconds":null,"sentLastHour":0,"failedLastHour":0,"serviceFailuresLastHour":0,"medianSeconds":null,"p95Seconds":null},"receiving":{"status":"unknown","creditedLastHour":0,"creditBacklog":null},"node":{"status":"unknown","latencyMs":null},"webhooks":{"status":"unknown","backlog":null}}}`
	statusAPI(t, 200, never)
	stdout, _, err = executeCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, want := range []string{"  ln.bot:    ⚠ Not measured yet", "None in progress", "Didn't answer", "Backlog unknown"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
}

func TestStatus_TooManyRequests(t *testing.T) {
	setupNoConfig(t)
	statusAPI(t, 429, `{"error":"Too many requests. Please try again later."}`)

	_, _, err := executeCmd("status")
	if err == nil || !strings.Contains(err.Error(), "too many requests") {
		t.Errorf("err = %v, want a too-many-requests error", err)
	}
}

func TestPay_PrintsStageChanges(t *testing.T) {
	setupConfig(t, testConfig())
	resetPayFlags(t)

	var mu sync.Mutex
	watches := 0
	events := []string{
		`event: timeout` + "\n" + `data: {"number":4,"status":"pending","amount":1000,"max_fee":20,"service_fee":3,"address":"x@y.com","stage":"retrying","status_detail":"Not sent yet; retrying for a few minutes.","retry_until":"2026-10-25T14:43:53+00:00"}`,
		`event: timeout` + "\n" + `data: {"number":4,"status":"pending","amount":1000,"max_fee":20,"service_fee":3,"address":"x@y.com","stage":"retrying","status_detail":"Not sent yet; retrying for a few minutes.","retry_until":"2026-10-25T14:43:53+00:00"}`,
		`event: timeout` + "\n" + `data: {"number":4,"status":"processing","amount":1000,"max_fee":20,"service_fee":3,"address":"x@y.com","stage":"in_flight","status_detail":"Sent; the Lightning network is completing it, usually within seconds.","retry_until":null}`,
		`event: settled` + "\n" + `data: {"number":4,"status":"settled","amount":1000,"max_fee":20,"service_fee":3,"actual_fee":2,"address":"x@y.com","stage":"settled"}`,
	}
	writeJSON := func(w http.ResponseWriter, body string) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/wallets/wal_main123/payments", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"number":4,"status":"pending","amount":1000,"maxFee":20,"serviceFee":3,"actualFee":null,"address":"x@y.com","stage":"queued","statusDetail":"Waiting to be sent.","retryUntil":null}`)
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments/4/events", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ev := events[min(watches, len(events)-1)]
		watches++
		mu.Unlock()
		if r.URL.Query().Get("timeout") != "10" {
			t.Errorf("watch timeout = %q, want 10", r.URL.Query().Get("timeout"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ev+"\n\n")
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments/4", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"number":4,"status":"settled","amount":1000,"maxFee":20,"serviceFee":3,"actualFee":2,"address":"x@y.com","stage":"settled","statusDetail":null,"retryUntil":null}`)
	})
	mux.HandleFunc("/v1/wallets/wal_main123", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"walletId":"wal_main123","name":"main","balance":77,"onHold":0,"available":77}`)
	})
	serve(t, mux)

	stdout, _, err := executeCmd("pay", "x@y.com", "--amount", "1000", "--yes")
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	want := []string{
		"  Queued: Waiting to be sent.\n",
		"  Waiting for settlement... (Ctrl+C to stop)\n",
		"  Retrying: Not sent yet; retrying for a few minutes. (until ",
		"  In flight: Sent; the Lightning network is completing it, usually within seconds.\n",
		"Payment settled",
		"Fees:       ₿5 (network ₿2, service ₿3)",
	}
	at := 0
	for _, line := range want {
		i := strings.Index(stdout[at:], line)
		if i < 0 {
			t.Fatalf("output lacks %q after position %d:\n%s", line, at, stdout)
		}
		at += i + len(line)
	}
	if n := strings.Count(stdout, "Retrying:"); n != 1 {
		t.Errorf("Retrying printed %d times, want once per change:\n%s", n, stdout)
	}
}

func TestPaymentList_ShowsStages(t *testing.T) {
	setupConfig(t, testConfig())
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/wallets/wal_main123/payments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"number":4,"status":"pending","amount":1000,"maxFee":20,"serviceFee":3,"address":"a@b.com","stage":"retrying"},
			{"number":3,"status":"processing","amount":500,"maxFee":20,"serviceFee":1,"address":"c@d.com","stage":"in_flight"},
			{"number":2,"status":"settled","amount":500,"maxFee":20,"serviceFee":1,"actualFee":2,"address":"e@f.com","stage":"settled"},
			{"number":1,"status":"processing","amount":200,"maxFee":20,"serviceFee":1,"address":"g@h.com"}
		]`)
	})
	serve(t, mux)

	stdout, _, err := executeCmd("payment", "list")
	if err != nil {
		t.Fatalf("payment list: %v", err)
	}
	for _, want := range []string{"Pending · retrying", "Processing · in flight", "#   2  Settled ", "#   1  Processing "} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Settled ·") || strings.Contains(stdout, "Processing · \n") {
		t.Errorf("stage shown where it shouldn't be:\n%s", stdout)
	}
}

func TestPay_WaitRunsOut(t *testing.T) {
	setupConfig(t, testConfig())
	resetPayFlags(t)
	oldWait, oldStage := watchTimeout, stageWatchTimeout
	watchTimeout, stageWatchTimeout = 1, 1
	t.Cleanup(func() { watchTimeout, stageWatchTimeout = oldWait, oldStage })

	inFlight := `{"number":4,"status":"processing","amount":1000,"maxFee":20,"serviceFee":3,"address":"x@y.com","stage":"in_flight","statusDetail":"Sent; the Lightning network is completing it, usually within seconds."}`
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/wallets/wal_main123/payments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, inFlight)
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments/4/events", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: timeout\ndata: "+`{"number":4,"status":"processing","amount":1000,"max_fee":20,"service_fee":3,"address":"x@y.com","stage":"in_flight","status_detail":"Sent; the Lightning network is completing it, usually within seconds."}`+"\n\n")
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments/4", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, inFlight)
	})
	serve(t, mux)

	stdout, _, err := executeCmd("pay", "x@y.com", "--amount", "1000", "--yes")
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	if !strings.HasSuffix(stdout, "  Still in progress. Check it with 'lnbot payment list'.\n  Sent; the Lightning network is completing it, usually within seconds.\n") {
		t.Errorf("unexpected ending:\n%s", stdout)
	}
	if strings.Contains(stdout, "retrying it") {
		t.Errorf("still promises retries:\n%s", stdout)
	}
}
