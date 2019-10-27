package openweather

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDegreeToCompass(t *testing.T) {

	tests := []struct {
		name    string
		degree  int
		want    string
		wantErr bool
		err     string
	}{
		{"0 is N", 0, "N", false, ""},
		{"30 is NNE", 30, "NNE", false, ""},
		{"99 is E", 99, "E", false, ""},
		{"123 is ESE", 123, "ESE", false, ""},
		{"315 is NW", 315, "NW", false, ""},
		{"360 is N", 360, "N", false, ""},

		// out of bounds errors
		{"-123 is out of bounds", -123, "", true, "degrees are out of bounds"},
		{"999 is out of bounds", 999, "", true, "degrees are out of bounds"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := DegreeToCompass(test.degree)

			if !test.wantErr {
				assert.Equal(test.want, got)
			} else {
				assert.EqualError(err, test.err)
			}
		})
	}
}
