package helm

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_cleanUpGenericMap(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "single level map",
			in: map[string]interface{}{
				"abc":    "xyz",
				"number": 5,
				"float":  1.5,
				"bool":   true,
				"array": []interface{}{
					"element",
				},
			},
			want: map[string]interface{}{
				"abc":    "xyz",
				"number": 5,
				"float":  1.5,
				"bool":   true,
				"array": []interface{}{
					"element",
				},
			},
		},
		{
			name: "nested map, string keys",
			in: map[string]interface{}{
				"nest": map[string]interface{}{
					"abc":    "xyz",
					"number": 5,
					"float":  1.5,
					"bool":   true,
					"array": []interface{}{
						"element",
					},
				},
			},
			want: map[string]interface{}{
				"nest": map[string]interface{}{
					"abc":    "xyz",
					"number": 5,
					"float":  1.5,
					"bool":   true,
					"array": []interface{}{
						"element",
					},
				},
			},
		},
		{
			name: "nested map, interface keys", // this is what would fail previously
			in: map[string]interface{}{
				"nest": map[interface{}]interface{}{
					"abc":    "xyz",
					"number": 5,
					"float":  1.5,
					"bool":   true,
					"array": []interface{}{
						"element",
					},
				},
			},
			want: map[string]interface{}{
				"nest": map[string]interface{}{
					"abc":    "xyz",
					"number": 5,
					"float":  1.5,
					"bool":   true,
					"array": []interface{}{
						"element",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(tt.want, cleanUpGenericMap(tt.in))
		})
	}
}
