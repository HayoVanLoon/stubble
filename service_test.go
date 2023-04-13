package stuble_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/HayyoVanLoon/stuble"
)

func TestHandler_GetResponse(t *testing.T) {
	getFoo := stuble.Rule{
		Method: http.MethodGet,
		Path:   "/foo",
		Resp:   stuble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooRegex := stuble.Rule{
		Method:    http.MethodGet,
		PathRegex: "/foo/.+",
		Resp:      stuble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooRegex, _ = stuble.InitRule(getFooRegex)
	getFooBodyRegex := stuble.Rule{
		BodyStringRegex: "(abc){2,4}",
		Resp:            stuble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooBodyRegex, _ = stuble.InitRule(getFooBodyRegex)
	getFooBodyJSON := stuble.Rule{
		BodyJSON: map[string]any{
			"foo": float64(123),
			"bar": "bla",
			"moo": stuble.Wildcard,
			"vla": []any{float64(2), float64(4)},
		},
		Resp: stuble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooBodyJSON, _ = stuble.InitRule(getFooBodyJSON)

	type fields struct {
		rules []stuble.Rule
	}
	type args struct {
		method string
		path   string
		body   io.Reader
	}
	type want struct {
		value stuble.Response
		err   require.ErrorAssertionFunc
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			"happy",
			fields{[]stuble.Rule{getFoo}},
			args{http.MethodGet, "/foo", nil},
			want{getFoo.Resp, require.NoError},
		},
		{
			"path regexp",
			fields{[]stuble.Rule{getFooRegex}},
			args{http.MethodGet, "/foo/bar", nil},
			want{getFooRegex.Resp, require.NoError},
		},
		{
			"body regexp",
			fields{[]stuble.Rule{getFooBodyRegex}},
			args{http.MethodPost, "/foo", bytes.NewReader([]byte("abcabcabc"))},
			want{getFooBodyRegex.Resp, require.NoError},
		},
		{
			"body JSON",
			fields{[]stuble.Rule{getFooBodyJSON}},
			args{http.MethodPost, "/foo", bytes.NewReader([]byte(`{"bar": "bla", "foo": 123, "moo": [1,2,3], "vla": [2,4]}`))},
			want{getFooBodyJSON.Resp, require.NoError},
		},
		{
			"not found",
			fields{[]stuble.Rule{getFoo}},
			args{http.MethodGet, "/moo", nil},
			want{stuble.NotFound, require.NoError},
		},
		{
			"no rules",
			fields{},
			args{http.MethodGet, "/foo", nil},
			want{stuble.NotFound, require.NoError},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := stuble.New(tt.fields.rules...)

			req, _ := http.NewRequest(tt.args.method, tt.args.path, tt.args.body)
			actual, err := h.GetResponse(req)
			tt.want.err(t, err)
			require.Equal(t, tt.want.value, actual)
		})
	}
}
