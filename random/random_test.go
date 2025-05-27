package random

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenRandomString(t *testing.T) {
	str32, err := String(32)
	require.NoError(t, err)
	str32Sec, err := String(32)
	require.NoError(t, err)

	assert.Len(t, str32, 32)
	assert.Len(t, str32Sec, 32)
	assert.NotEqual(t, str32, str32Sec)

	str64, err := String(64)
	require.NoError(t, err)
	str64Sec, err := String(64)
	require.NoError(t, err)
	assert.Len(t, str64, 64)
	assert.Len(t, str64Sec, 64)
	assert.NotEqual(t, str64, str64Sec)
}
