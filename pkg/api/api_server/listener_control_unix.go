//go:build !windows

package api_server

import "syscall"

func dualStackControl(_, _ string, c syscall.RawConn) error {
	var innerErr error
	err := c.Control(func(fd uintptr) {
		innerErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY, 0)
	})
	if err != nil {
		return err
	}
	return innerErr
}
