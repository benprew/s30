//go:build !pprof

package pprofutil

import "net"

func Start(string, func(string, ...any)) (net.Listener, error) {
	return nil, nil
}
