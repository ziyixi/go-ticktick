package ticktick

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	// string contains
	assert := assert.New(t)
	assert.True(Contains([]string{"a", "b"}, "a"))
	assert.False(Contains([]string{"a", "b"}, "aaa"))

	// int contains
	assert.True(Contains([]int64{123, 456}, 123))
	assert.False(Contains([]int64{123, 456}, 1233))
}
