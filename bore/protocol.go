package bore

import (
	"bytes"
	"encoding/json"
	"fmt"

	"io"

	"github.com/google/uuid"
)

type ClientHello struct {
	Port int `json:"Hello"`
}

type ClientAuthenticate struct {
	Authenticate string
}

type ClientAccept struct {
	Accept uuid.UUID
}

type ServerChallenge struct {
	Challenge uuid.UUID
}

type ServerHello struct {
	Port int `json:"Hello"`
}

type ServerConnection struct {
	Connection uuid.UUID
}

type ServerError struct {
	Error string
}

type ServerHertbeat struct {
}

func (m ServerHertbeat) MarshalJSON() ([]byte, error) {
	return []byte("\"Heartbeat\""), nil
}

func (m *ServerHertbeat) UnmarshalJSON(data []byte) error {
	s := ""
	json.Unmarshal(data, &s)
	if s != "Heartbeat" {
		return fmt.Errorf("not Heartbeat")
	}
	return nil
}

const CONTROL_PORT = 7835

func parseClientMessage(data []byte) (any, error) {
	hash := map[string]any{}
	err := json.Unmarshal(data, &hash)
	if err != nil {
		return nil, err
	}

	if accept, ok := hash["Accept"]; ok {
		if accept, ok := accept.(string); ok {
			id, err := uuid.Parse(accept)
			if err != nil {
				return nil, err
			}
			return ClientAccept{Accept: id}, nil
		}
	}

	if port, ok := hash["Hello"]; ok {
		if port, ok := port.(float64); ok {
			if port >= 0 && port <= 65535 {
				return ClientHello{Port: int(port)}, nil
			}
		}
	}

	if auth, ok := hash["Authenticate"]; ok {
		if auth, ok := auth.(string); ok {
			return ClientAuthenticate{Authenticate: auth}, nil
		}
	}

	return nil, fmt.Errorf("failed to parse client message: %s", string(data))
}

func SendJson(w io.Writer, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, 0)

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	return nil
}

type MessageParser = func([]byte) (any, error)

func RecvJson(r BufferedConn, parse MessageParser) (any, error) {
	data, err := r.ReadBytes(0)
	if err != nil {
		return nil, err
	}
	// remove null byte
	data = data[:len(data)-1]

	return parse(data)
}

func RecvClientMessage(r BufferedConn) (any, error) {
	return RecvJson(r, parseClientMessage)
}

func RecvServerMessage(r BufferedConn) (any, error) {
	return RecvJson(r, parseServerMessage)
}

func parseServerMessage(data []byte) (any, error) {
	// heartbeat is a string
	if bytes.Equal([]byte("\"Heartbeat\""), data) {
		return ServerHertbeat{}, nil
	}

	hash := map[string]any{}
	err := json.Unmarshal(data, &hash)
	if err != nil {
		return nil, err
	}

	if port, ok := hash["Hello"]; ok {
		if port, ok := port.(float64); ok {
			if port >= 0 && port <= 65535 {
				return ServerHello{Port: int(port)}, nil
			}
		}
	}

	if conn, ok := hash["Connection"]; ok {
		if conn, ok := conn.(string); ok {
			id, err := uuid.Parse(conn)
			if err != nil {
				return nil, err
			}
			return ServerConnection{Connection: id}, nil
		}
	}

	if challenge, ok := hash["Challenge"]; ok {
		if challenge, ok := challenge.(string); ok {
			id, err := uuid.Parse(challenge)
			if err != nil {
				return nil, err
			}
			return ServerChallenge{Challenge: id}, nil
		}
	}

	if errmsg, ok := hash["Error"]; ok {
		if errmsg, ok := errmsg.(string); ok {
			return ServerError{Error: errmsg}, nil
		}
	}

	return nil, fmt.Errorf("failed to parse server message: %s", string(data))
}
