package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	var randReader RandReader
	buf := make([]byte, 100)
	n, err := randReader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 100, n)
}
