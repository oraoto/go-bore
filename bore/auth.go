package bore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/google/uuid"
)

type Authenticator struct {
	secret []byte
}


func NewAuthenticator(secret string) *Authenticator {
	hashed_secret := sha256.Sum256([]byte(secret))
	return &Authenticator{
		secret: hashed_secret[:],
	}
}

func (a *Authenticator) newHmac() hash.Hash {
	return hmac.New(sha256.New, a.secret)
}

func (a *Authenticator) Answer(challenge uuid.UUID) string {
	h := a.newHmac()
	h.Write(challenge[:])
	mac := h.Sum(nil)
	return hex.EncodeToString(mac)
}

func (a *Authenticator) Validate(challenge uuid.UUID, tagstr string) bool {
	tag, err := hex.DecodeString(tagstr)
	if err != nil {
		return false
	}
	h := a.newHmac()
	h.Write(challenge[:])
	return hmac.Equal(tag, h.Sum(nil))
}

func (a *Authenticator) ServerHandshake(conn BufferedConn) error {
	challenge := uuid.New()
	if err := SendJson(conn, ServerChallenge{Challenge: challenge}); err != nil {
		return err
	}
	msg, err := RecvClientMessage(conn)
	if err != nil {
		return err
	}
	if auth, ok := msg.(ClientAuthenticate); ok {
		if a.Validate(challenge, auth.Authenticate) {
			return nil
		}
	}
	return fmt.Errorf("authencate failed")
}

func (a *Authenticator) ClientHandshake(conn BufferedConn) error {
	msg, err := RecvServerMessage(conn)
	if err != nil {
		return err
	}

	if challenge, ok := msg.(ServerChallenge); ok {
		tag := a.Answer(challenge.Challenge)
		err = SendJson(conn, ClientAuthenticate{Authenticate: tag})
		if err != nil {
			return err
		}
	}

	return nil
}
