package climate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFileV3(t *testing.T) {
	_, err := LoadFileV3("api.yaml")
	assert.NoError(t, err)
}
