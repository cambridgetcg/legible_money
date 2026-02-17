//go:build openclaw

package devnet

import (
	"fmt"
	"net"
)

// AllocatePorts returns n unique available TCP ports by binding to :0,
// reading the assigned port, then closing the listener.
func AllocatePorts(n int) ([]int, error) {
	ports := make([]int, n)
	listeners := make([]net.Listener, n)

	for i := 0; i < n; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			// Close any already-opened listeners
			for j := 0; j < i; j++ {
				listeners[j].Close()
			}
			return nil, fmt.Errorf("allocating port %d/%d: %w", i+1, n, err)
		}
		listeners[i] = l
		ports[i] = l.Addr().(*net.TCPAddr).Port
	}

	// Close all listeners to release ports for use
	for _, l := range listeners {
		l.Close()
	}

	return ports, nil
}
