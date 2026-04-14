package pprofutil

import (
	"errors"
	"net"
	"net/http"
	_ "net/http/pprof"
)

var (
	listen = net.Listen
	serve  = http.Serve
)

func Start(addr string, logf func(string, ...any)) (net.Listener, error) {
	if addr == "" {
		return nil, nil
	}

	ln, err := listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		err := serve(ln, nil)
		if err != nil && !errors.Is(err, net.ErrClosed) && logf != nil {
			logf("pprof server stopped: %v", err)
		}
	}()

	if logf != nil {
		logf("pprof listening on http://%s/debug/pprof/", ln.Addr().String())
	}

	return ln, nil
}
