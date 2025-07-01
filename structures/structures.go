package structures

import (
	"net"
)

type Client struct {
	Conn     net.Conn
	Name     string
	Password string
	Role     string
}
