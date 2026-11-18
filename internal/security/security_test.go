// Package security contains unit tests for the cryptographic core.
//
// The tests use a slog.Handler capture to verify that secrets never reach
// the log.
package security

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
)

// slogCaptureHandler is a slog.Handler that stores every log record in the
// fully formatted form, so tests can assert that secrets do not reach the
// log.
type slogCaptureHandler struct {
	mu      sync.Mutex
	records []string
}

// Enabled reports that the handler processes every record at every level.
func (h *slogCaptureHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle stores a fully formatted copy of the record (message and all
// attributes) via a text handler.
func (h *slogCaptureHandler) Handle(ctx context.Context, r slog.Record) error {
	var buf bytes.Buffer
	if err := slog.NewTextHandler(&buf, nil).Handle(ctx, r); err != nil {
		return err
	}
	h.mu.Lock()
	h.records = append(h.records, buf.String())
	h.mu.Unlock()
	return nil
}

// WithAttrs ignores attributes in tests.
func (h *slogCaptureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

// WithGroup ignores groups in tests.
func (h *slogCaptureHandler) WithGroup(_ string) slog.Handler {
	return h
}

// containsSecret reports whether any of secrets appears in the captured log
// output.
func (h *slogCaptureHandler) containsSecret(secrets ...string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, line := range h.records {
		for _, s := range secrets {
			if strings.Contains(line, s) {
				return true
			}
		}
	}
	return false
}

// withHandler runs fn with h installed as the default slog logger and
// restores the previous default after it returns.
func withHandler(h slog.Handler, fn func()) {
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(prev)
	fn()
}
