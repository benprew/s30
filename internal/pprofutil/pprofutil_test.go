package pprofutil

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestStartDisabled(t *testing.T) {
	t.Parallel()

	ln, err := Start("", nil)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if ln != nil {
		t.Fatalf("expected nil listener when pprof is disabled")
	}
}

func TestStartStartsServerAndLogsAddress(t *testing.T) {
	t.Parallel()

	prevListen := listen
	prevServe := serve
	t.Cleanup(func() {
		listen = prevListen
		serve = prevServe
	})

	fakeLn := &stubListener{addr: stubAddr("127.0.0.1:6060")}
	listen = func(network, addr string) (net.Listener, error) {
		if network != "tcp" {
			t.Fatalf("unexpected network %q", network)
		}
		if addr != "127.0.0.1:6060" {
			t.Fatalf("unexpected addr %q", addr)
		}
		return fakeLn, nil
	}

	served := make(chan struct{}, 1)
	serve = func(ln net.Listener, handler http.Handler) error {
		if ln != fakeLn {
			t.Fatalf("unexpected listener passed to serve")
		}
		if handler != nil {
			t.Fatalf("expected nil handler, got %#v", handler)
		}
		served <- struct{}{}
		return net.ErrClosed
	}

	var logs []string
	ln, err := Start("127.0.0.1:6060", func(format string, args ...any) {
		logs = append(logs, format)
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if ln != fakeLn {
		t.Fatalf("expected returned listener to match fake listener")
	}
	if len(logs) != 1 || logs[0] != "pprof listening on http://%s/debug/pprof/" {
		t.Fatalf("unexpected log messages: %#v", logs)
	}

	select {
	case <-served:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected serve to be called")
	}
}

func TestStartReturnsListenError(t *testing.T) {
	t.Parallel()

	prevListen := listen
	t.Cleanup(func() {
		listen = prevListen
	})

	wantErr := errors.New("boom")
	listen = func(network, addr string) (net.Listener, error) {
		return nil, wantErr
	}

	ln, err := Start("127.0.0.1:6060", nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if ln != nil {
		t.Fatalf("expected nil listener on error")
	}
}

type stubListener struct {
	addr net.Addr
}

func (l *stubListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (l *stubListener) Close() error              { return nil }
func (l *stubListener) Addr() net.Addr            { return l.addr }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }
