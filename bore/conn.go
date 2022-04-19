package bore

import (
	"bufio"
	"io"
	"net"
)

// Buffered read and unbuffered write
//
// We need buffered ReadBytes to decode messages
type BufferedConn interface {
	io.ReadWriteCloser
	ReadBytes(delim byte) ([]byte, error)
}

type BufConn struct {
	// keep bufio.Reader first, delegate Read() to it
	*bufio.Reader
	*net.TCPConn
}

var _ BufferedConn = &BufConn{}

func NewBufConn(conn *net.TCPConn) *BufConn {
	return &BufConn{
		TCPConn: conn,
		Reader:  bufio.NewReader(conn),
	}
}
