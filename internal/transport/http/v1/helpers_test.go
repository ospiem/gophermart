package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isValidByLuhnAlgo(t *testing.T) {
	testCases := []struct {
		name    string
		numbers []int
		want    bool
	}{
		{name: "valid numbers 1",
			numbers: []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1, 3},
			want:    true,
		}, {name: "valid numbers 2",
			numbers: []int{7, 5, 9, 6, 7, 3, 9, 3, 7, 1, 3},
			want:    true,
		}, {name: "valid numbers 3",
			numbers: []int{6, 2, 8, 2, 5, 8, 0, 3, 1, 6, 0, 3, 1, 1, 4, 4, 6, 8, 2},
			want:    true,
		}, {name: "valid numbers 4",
			numbers: []int{1, 2, 3, 1},
			want:    true,
		}, {name: "invalid numbers 1",
			numbers: []int{0, 9, 1, 2, 3, 7, 4, 5, 1, 6, 7, 2},
			want:    false,
		}, {name: "invalid numbers 2",
			numbers: []int{9, 3, 4, 5, 7, 8, 9, 2, 3, 7},
			want:    false,
		}, {name: "invalid numbers 3",
			numbers: []int{1, 2, 3, 4},
			want:    false,
		}, {name: "invalid numbers 4",
			numbers: []int{9, 3, 2, 5, 8, 9, 7, 4, 3, 1, 0, 2, 9, 4, 3, 2, 5, 4, 3, 7, 8, 5},
			want:    false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidByLuhnAlgo(tc.numbers)
			assert.Equal(t, tc.want, result)
		})
	}
}
