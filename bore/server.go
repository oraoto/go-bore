package bore

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Server struct {
	minPort int
	auth    *Authenticator
	conns   sync.Map
}

func NewServer(minPort int, secret string) *Server {
	var auth *Authenticator
	if secret != "" {
		auth = NewAuthenticator(secret)
	}

	return &Server{
		minPort: minPort,
		auth:    auth,
		conns:   sync.Map{},
	}
}

func (s *Server) Listen() error {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: CONTROL_PORT})
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			return fmt.Errorf("accept error: %w", err)
		}
		addr := conn.RemoteAddr().String()
		log.Info().Str("control", addr).Msg("incoming connection")
		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Error().Str("control", addr).Msgf("panic %v", err)
				}
			}()
			err := s.handleConnection(NewBufConn(conn))
			if err != nil {
				log.Warn().Err(err).Str("control", addr).Msg("connectiion exited with error")
			} else {
				log.Info().Str("control", addr).Msg("conection exited")
			}
		}()

	}
}

func (s *Server) handleConnection(conn *BufConn) error {
	defer conn.Close()

	if s.auth != nil {
		err := s.auth.ServerHandshake(conn)
		if err != nil {
			if err := SendJson(conn, ServerError{err.Error()}); err != nil {
				return err
			}
			return nil
		}
	}

	for {
		msg, err := RecvClientMessage(conn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch msg.(type) {
		case ClientHello:
			hello := msg.(ClientHello)
			// validate port range
			if hello.Port != 0 && hello.Port < s.minPort {
				log.Warn().Int("port", hello.Port).Msg("client port number too low")
				if err := SendJson(conn, ServerError{"client port number too low"}); err != nil {
					return err
				}
				return nil
			}
			log.Info().Int("port", hello.Port).Msg("new client")

			// listen on new port
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: hello.Port})
			if err != nil {
				log.Error().Int("port", hello.Port).Err(err).Msg("could not bind to local port")
				if err := SendJson(conn, ServerError{"port already in use"}); err != nil {
					return err
				}
				return nil
			}
			defer listener.Close()

			// send listen port back to client
			localPort := listener.Addr().(*net.TCPAddr).Port
			if err := SendJson(conn, ServerHello{Port: localPort}); err != nil {
				return err
			}

			// accept user connection
			for {
				// when hertbeat fails, close listener
				if err := SendJson(conn, ServerHertbeat{}); err != nil {
					return err
				}

				listener.SetDeadline(time.Now().Add(500 * time.Millisecond))
				client, err := listener.AcceptTCP()
				if err != nil {
					continue
				}
				log.Info().Str("addr", client.RemoteAddr().String()).Msg("new connection")

				id := uuid.New()
				s.conns.Store(id, client)

				// close connection if not connected in 10 seconds
				time.AfterFunc(10*time.Second, func() {
					conn, ok := s.conns.LoadAndDelete(id)
					if ok {
						log.Info().Str("id", id.String()).Msg("removed stale connection")
						conn.(*net.TCPConn).Close()
					}
				})

				// tell bore client to accept connection
				if err := SendJson(conn, ServerConnection{Connection: id}); err != nil {
					log.Error().Err(err).Msg("server acception connection")
					return err
				}
			}
		case ClientAccept:
			accept := msg.(ClientAccept)
			log.Info().Str("id", accept.Accept.String()).Msg("forwarding connection")

			userConn, ok := s.conns.LoadAndDelete(accept.Accept)
			if !ok {
				log.Warn().Str("id", accept.Accept.String()).Msg("missing connectiion")
			} else {
				userConn := userConn.(*net.TCPConn)
				// send buffered bytes to user
				buffered := conn.Buffered()
				if buffered > 0 {
					log.Debug().Int("buffered", buffered).Msg("consume buffered bytes")
					buf := make([]byte, buffered)
					conn.Read(buf)
					userConn.Write(buf)
				}
				proxy(conn, userConn)
			}
			return nil
		case ClientAuthenticate:
			log.Error().Msg("unexpected authenticate msg")
			return nil
		}
	}
}

func proxy(conn1, conn2 io.ReadWriter) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	stream := func(c1, c2 io.ReadWriter) {
		io.Copy(c1, c2)
		wg.Done()
	}
	go stream(conn1, conn2)
	go stream(conn2, conn1)

	wg.Wait()
}
