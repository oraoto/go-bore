package bore_test

import (
	"encoding/json"
	"testing"

	"github.com/oraoto/go-bore/bore"
	"github.com/stretchr/testify/assert"
)


func TestServerHeartSerilize(t *testing.T) {
	msg := bore.ServerHertbeat{}

	data, _ := json.Marshal(msg)
	str := string(data)

	assert.Equal(t, str, "\"Heartbeat\"")

	err := json.Unmarshal(data, &msg)
	assert.Nil(t, err)

	err = json.Unmarshal([]byte("whatever"), &msg)
	assert.Error(t, err)
}
