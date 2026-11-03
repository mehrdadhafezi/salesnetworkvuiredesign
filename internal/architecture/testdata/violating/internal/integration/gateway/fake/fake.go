// Package fake is a VIOLATING Slice 12 fixture.
package fake

import "net"

// NewSession would log a customer in after a callback (D-26).
func NewSession() {}

func dial() { _, _ = net.Dial("tcp", "gateway.invalid:443") }
