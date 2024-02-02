package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isValidByLuhnAlgo(t *testing.T) {
	testCases := []struct {
		name      string
		orderNum  string
		wantError bool
	}{
		{name: "valid numbers 1",
			orderNum:  "79927398713",
			wantError: false,
		}, {name: "valid numbers 2",
			orderNum:  "75967393713",
			wantError: false,
		}, {name: "valid numbers 3",
			orderNum:  "6282580316031144682",
			wantError: false,
		}, {name: "valid numbers 4",
			orderNum:  "1231",
			wantError: false,
		}, {name: "invalid numbers 1",
			orderNum:  "091237451672",
			wantError: true,
		}, {name: "invalid numbers 2",
			orderNum:  "9345789237",
			wantError: true,
		}, {name: "invalid numbers 3",
			orderNum:  "1234",
			wantError: true,
		}, {name: "invalid numbers 4",
			orderNum:  "9325897431029432543785",
			wantError: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validByLuhnAlgo(tc.orderNum)
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
