//go:build !pprof

package pprofutil

import "testing"

func TestStartIsNoOpWithoutPprofBuildTag(t *testing.T) {
	ln, err := Start("127.0.0.1:6060", t.Logf)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if ln != nil {
		_ = ln.Close()
		t.Fatal("Start returned a listener without the pprof build tag")
	}
}
