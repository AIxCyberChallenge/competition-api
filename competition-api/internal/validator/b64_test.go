package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatchSize(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		assert.True(t, ValidatePatchSize(len(base64String((1<<10)*100))), "max size should work")
	})

	t.Run("ValidSmall", func(t *testing.T) {
		assert.True(t, ValidatePatchSize(len(base64String(10))), "small size should work")
	})

	t.Run("Invalid", func(t *testing.T) {
		assert.False(t, ValidatePatchSize(len(base64String((1<<10)*101))), "too big")
	})
}

func TestTriggerSize(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		assert.True(t, ValidateTriggerSize(len(base64String(1<<21))), "max size should work")
	})

	t.Run("ValidSmall", func(t *testing.T) {
		assert.True(t, ValidateTriggerSize(len(base64String(10))), "small size should work")
	})

	t.Run("Invalid", func(t *testing.T) {
		assert.False(t, ValidateTriggerSize(len(base64String((1<<21)+100))), "too big")
	})
}
