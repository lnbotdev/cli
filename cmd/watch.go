package cmd

import (
	"context"
	"time"

	lnbot "github.com/lnbotdev/go-sdk"
)

// lateSettleWindow is how long an expired invoice is still watched: one paid
// just before it expired can settle afterwards.
const lateSettleWindow = time.Minute

// pollInterval is how often an invoice is read directly when its event
// stream isn't available or it has expired.
const pollInterval = 3 * time.Second

// watchTimeout is the longest wait the API allows on one event stream.
var watchTimeout = lnbot.Ptr(300)

// waitPayment waits on the payment's event stream until it settles, fails or
// the stream ends, then reads the payment as it is now. The stream's payload
// names fields in snake_case (actual_fee, failure_reason), which the SDK's
// Payment doesn't decode, so only the read carries the fees and the reason.
func waitPayment(ctx context.Context, w *lnbot.WalletHandle, number int) (*lnbot.Payment, error) {
	if _, err := watchPayment(ctx, w, number); err != nil {
		return nil, err
	}
	return w.Payments.Get(ctx, number)
}

// watchPayment waits on the payment's event stream and returns the payment
// from its settled or failed event, or nil if the stream ended without one.
// Only its status is reliable: see waitPayment.
func watchPayment(ctx context.Context, w *lnbot.WalletHandle, number int) (*lnbot.Payment, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, errs := w.Payments.Watch(ctx, number, watchTimeout)
	for events != nil || errs != nil {
		select {
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			return &ev.Data, nil
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

// watchInvoice waits on the invoice's event stream and returns the invoice
// from its settled or expired event, or nil if the stream ended without one.
func watchInvoice(ctx context.Context, w *lnbot.WalletHandle, number int) (*lnbot.Invoice, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, errs := w.Invoices.Watch(ctx, number, watchTimeout)
	for events != nil || errs != nil {
		select {
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			return &ev.Data, nil
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

// waitForInvoice follows an invoice until it settles, or until it has been
// expired for lateSettleWindow. onExpired runs once, when it first shows as
// expired. If the event stream fails, it reads the invoice directly instead.
// It always returns the invoice as read, not the event's payload, which uses
// snake_case field names the SDK doesn't decode.
func waitForInvoice(ctx context.Context, w *lnbot.WalletHandle, number int, onExpired func()) (*lnbot.Invoice, error) {
	var expiredAt time.Time
	poll := false

	for {
		if poll || !expiredAt.IsZero() {
			if err := sleepCtx(ctx, pollInterval); err != nil {
				return nil, err
			}
		} else {
			if _, err := watchInvoice(ctx, w, number); err != nil {
				poll = true
			}
		}

		inv, err := w.Invoices.Get(ctx, number)
		if err != nil {
			return nil, err
		}

		switch inv.Status {
		case "settled":
			return inv, nil
		case "expired":
			if expiredAt.IsZero() {
				expiredAt = time.Now()
				if onExpired != nil {
					onExpired()
				}
			} else if time.Since(expiredAt) >= lateSettleWindow {
				return inv, nil
			}
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
