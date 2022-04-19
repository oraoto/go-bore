package bore

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Client struct {
	remoteHost string
	remotePort int
	localPort int
	auth *Authenticator
	conns sync.Map
}

func NewClient(remoteHost string, remotePort int, localPort int, secret string) *Client {
	var auth *Authenticator
	if secret != "" {
		auth = NewAuthenticator(secret)
	}
	return &Client{
		remoteHost: remoteHost,
		remotePort: remotePort,
		localPort: localPort,
		auth: auth,
		conns: sync.Map{},
	}
}

func (c *Client) Handshake() (*BufConn, error) {
	// connect to remote server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.remoteHost, CONTROL_PORT))
	if err != nil {
		return nil, fmt.Errorf("connect to remote server failed: %w", err)
	}

	bufconn := NewBufConn(conn.(*net.TCPConn))

	if c.auth != nil {
		err := c.auth.ClientHandshake(bufconn)
		if err != nil {
			bufconn.Close()
			return nil, err
		}
	}

	return bufconn, nil
}

func (c *Client) Start() error {
	conn, err := c.Handshake()
	if err != nil {
		return err
	}
	defer conn.Close()

	// send client hello
	if err := SendJson(conn, ClientHello{Port: c.remotePort}); err != nil {
		return err
	}

	for {
		msg, err := RecvServerMessage(conn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch msg.(type) {
		case ServerHello:
			hello := msg.(ServerHello)
			fmt.Printf("listening at %s:%d\n", c.remoteHost, hello.Port)
		case ServerConnection:
			go c.handleConnection(msg.(ServerConnection))
		case ServerError:
			error := msg.(ServerError)
			fmt.Printf("server error: %s\n", error.Error)
			return nil
		case ServerHertbeat:
			continue
		default:
			return nil
		}
	}
}

func (c *Client) handleConnection(msg ServerConnection) {
	serverConn, err := c.Handshake()
	if err != nil {
		fmt.Printf("error: %w", err)
		return
	}
	defer serverConn.Close()

	if err := SendJson(serverConn, ClientAccept{Accept: msg.Connection}); err != nil {
		fmt.Printf("error: %w", err)
		return
	}

	localConn, err := net.DialTimeout("tcp", fmt.Sprintf("0.0.0.0:%d", c.localPort), 3 * time.Second)
	if err != nil {
		fmt.Printf("can't connect to local service: %w", err)
		return
	}
	defer localConn.Close()

	buffered := serverConn.Buffered()
	if buffered > 0 {
		buf := make([]byte, buffered)
		serverConn.Read(buf)
		localConn.Write(buf)
	}
	proxy(serverConn, localConn)
}
