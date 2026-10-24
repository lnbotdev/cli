package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// redirect sends every request the SDK makes to a test server.
type redirect struct{ to *url.URL }

func (r redirect) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.to.Scheme
	req.URL.Host = r.to.Host
	return http.DefaultTransport.RoundTrip(req)
}

// mockAPI serves a payment that is pending when created and settles on its
// event stream. The event's payload is snake_case, as the API sends it; the
// REST read is camelCase.
func mockAPI(t *testing.T) {
	t.Helper()
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/v1/wallets/wal_main123", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"walletId": "wal_main123", "name": "main", "balance": 77, "onHold": 0, "available": 77})
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"number": 1, "status": "pending", "amount": 20, "maxFee": 5, "serviceFee": 1, "actualFee": nil, "address": "bub@bub.gg"})
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments/1/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `event: settled`+"\n"+`data: {"number":1,"status":"settled","amount":20,"max_fee":5,"service_fee":1,"actual_fee":3,"address":"bub@bub.gg"}`+"\n\n")
	})
	mux.HandleFunc("/v1/wallets/wal_main123/payments/1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"number": 1, "status": "settled", "amount": 20, "maxFee": 5, "serviceFee": 1, "actualFee": 3, "address": "bub@bub.gg"})
	})
	mux.HandleFunc("/v1/wallets/wal_main123/invoices", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"number": 7, "status": "pending", "amount": 1000, "bolt11": "lnbc10u1p"})
	})
	mux.HandleFunc("/v1/wallets/wal_main123/invoices/7/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `event: settled`+"\n"+`data: {"number":7,"status":"settled","amount":1000,"tx_number":4}`+"\n\n")
	})
	mux.HandleFunc("/v1/wallets/wal_main123/invoices/7", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"number": 7, "status": "settled", "amount": 1000, "bolt11": "lnbc10u1p", "txNumber": 4})
	})

	srv := httptest.NewServer(mux)
	u, _ := url.Parse(srv.URL)
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = redirect{u}
	t.Cleanup(func() {
		http.DefaultClient.Transport = old
		srv.Close()
	})
}

func TestPay_SettledAfterWaitShowsFees(t *testing.T) {
	setupConfig(t, testConfig())
	resetPayFlags(t)
	mockAPI(t)

	stdout, _, err := executeCmd("pay", "bub@bub.gg", "--amount", "20", "--yes")
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	for _, want := range []string{"Payment settled", "Fees:       ₿4 (network ₿3, service ₿1)", "Total:      ₿24"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
}

func TestPay_SettledAfterWaitJSON(t *testing.T) {
	setupConfig(t, testConfig())
	resetPayFlags(t)
	mockAPI(t)

	stdout, _, err := executeCmd("pay", "bub@bub.gg", "--amount", "20", "--yes", "--json")
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	var p struct {
		Status     string `json:"status"`
		ActualFee  *int64 `json:"actualFee"`
		ServiceFee int64  `json:"serviceFee"`
	}
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("decoding %q: %v", stdout, err)
	}
	if p.Status != "settled" || p.ActualFee == nil || *p.ActualFee != 3 || p.ServiceFee != 1 {
		t.Errorf("got status %q, actualFee %v, serviceFee %d", p.Status, p.ActualFee, p.ServiceFee)
	}
}

func TestInvoiceCreate_SettledJSONIsTheRESTInvoice(t *testing.T) {
	setupConfig(t, testConfig())
	mockAPI(t)
	t.Cleanup(func() { invoiceCreateCmd.Flags().Set("amount", "0") })

	stdout, _, err := executeCmd("invoice", "create", "--amount", "1000", "--json")
	if err != nil {
		t.Fatalf("invoice create: %v", err)
	}
	if !strings.Contains(stdout, `"txNumber":4`) {
		t.Errorf("expected the REST invoice with txNumber 4, got %s", stdout)
	}
}
