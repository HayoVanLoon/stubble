package examples

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/HayoVanLoon/stubble"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExample(t *testing.T) {
	dec := json.NewDecoder(bytes.NewReader(file))
	var r1 stubble.Rule
	parseErr := dec.Decode(&r1)
	require.NoError(t, parseErr)
	var r2 stubble.Rule
	parseErr = dec.Decode(&r2)
	require.NoError(t, parseErr)
	var r3 stubble.Rule
	parseErr = dec.Decode(&r3)
	require.NoError(t, parseErr)
	var r4 stubble.Rule
	parseErr = dec.Decode(&r4)
	require.NoError(t, parseErr)
	require.False(t, dec.More())

	tests := []struct {
		args stubble.Rule
		want stubble.Rule
	}{
		{r1, stubble.Rule{
			Name:   "get_fallback",
			Method: "GET",
			Path:   "/",
			Response: stubble.Response{
				StatusCode: 200,
				BodyString: "hello world",
			},
		}},
		{r2, stubble.Rule{
			Name:   "get_foo",
			Method: "GET",
			Path:   "/foo",
			Response: stubble.Response{
				StatusCode: 200,
				BodyString: "hello foo",
			},
		}},
		{r3, stubble.Rule{
			Name: "match_body",
			BodyJSON: map[string]any{
				"number": float64(123),
				"text":   ".*",
			},
			Response: stubble.Response{
				StatusCode: 200,
				BodyString: "hello body",
			},
		}},
		{r4, stubble.Rule{
			Name: "match_other_body",
			BodyJSON: map[string]any{
				"answer": float64(42),
			},
			Response: stubble.Response{
				StatusCode: 200,
				BodyJSON: map[string]any{
					"foo": "bar",
					"bla": "vla",
				},
			},
		}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("item-%d", i), func(t *testing.T) {
			require.Equal(t, tt.want, tt.args)
		})
	}
}

//go:embed stuble-example.ndjson
var file []byte
