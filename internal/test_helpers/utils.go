package testhelpers

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	bytesToMB = 1024 * 1024
)

func GenerateMBData(t *testing.T, mbNum float64) []byte {
	data := make([]byte, int(mbNum*float64(bytesToMB)))
	_, err := rand.Read(data)
	require.NoError(t, err)
	return data
}
