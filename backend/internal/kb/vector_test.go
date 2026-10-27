package kb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorRoundTrip(t *testing.T) {
	in := []float32{0.25, -1.5, 3.0}
	b := EncodeVector(in)
	assert.Len(t, b, 12)
	out, err := DecodeVector(b, 3)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestDecodeVectorRejectsWrongLength(t *testing.T) {
	_, err := DecodeVector(make([]byte, 8), 3)
	require.Error(t, err)
}
