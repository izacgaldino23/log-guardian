package unix

import (
	"net"
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type Conn interface {
	Close() error
	SetReadDeadline(t time.Time) error
	Read(b []byte) (n int, err error)
}

type NewNetConnectionWithTimeout func(network string, address string, timeout time.Duration) (net.Conn, error)

type ConnectionFactory func(network, address string, timeout time.Duration) (Conn, error)
