package stubble_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/HayoVanLoon/stubble"
)

func TestRule_Match(t *testing.T) {
	getFoo := stubble.Rule{
		Method:   http.MethodGet,
		Path:     "/foo",
		Response: stubble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooRegex := stubble.Rule{
		Method:    http.MethodGet,
		PathRegex: "/foo/.+",
		Response:  stubble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooRegex, _ = stubble.InitRule(getFooRegex)
	getFooBodyRegex := stubble.Rule{
		BodyStringRegex: "(abc){2,4}",
		Response:        stubble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooBodyRegex, _ = stubble.InitRule(getFooBodyRegex)
	getFooBodyJSON := stubble.Rule{
		BodyJSON: map[string]any{
			"foo": float64(123),
			"bar": "bla",
			"moo": stubble.Wildcard,
			"vla": []any{float64(2), float64(4)},
		},
		Response: stubble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getFooBodyJSON, _ = stubble.InitRule(getFooBodyJSON)
	getParams := stubble.Rule{
		Method:   http.MethodGet,
		Path:     "/foo",
		Params:   map[string][]string{"a": {"b"}, "x": {"y", "z"}},
		Response: stubble.Response{StatusCode: http.StatusOK, BodyString: "bar"},
	}
	getParams, _ = stubble.InitRule(getParams)

	type fields struct {
		rule stubble.Rule
	}
	type args struct {
		method  string
		path    string
		params  map[string][]string
		headers map[string][]string
		body    []byte
	}
	type want struct {
		value int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			"happy",
			fields{getFoo},
			args{http.MethodGet, "/foo", nil, nil, nil},
			want{2},
		},
		{
			"path regexp",
			fields{getFooRegex},
			args{http.MethodGet, "/foo/bar", nil, nil, nil},
			want{2},
		},
		{
			"body regexp",
			fields{getFooBodyRegex},
			args{http.MethodPost, "/foo", nil, nil, []byte("abcabcabc")},
			want{1},
		},
		{
			"body JSON",
			fields{getFooBodyJSON},
			args{http.MethodPost, "/foo", nil, nil, []byte(`{"bar": "bla", "foo": 123, "moo": [1,2,3], "vla": [2,4]}`)},
			want{1},
		},
		{
			"params",
			fields{getParams},
			args{
				http.MethodGet,
				"/foo",
				map[string][]string{"x": {"z", "y", "bla"}, "a": {"b"}},
				nil,
				nil,
			},
			want{3},
		},
		{
			"not found",
			fields{getFoo},
			args{http.MethodGet, "/moo", nil, nil, nil},
			want{-999},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.args.path
			if tt.args.params != nil {
				p += "?" + url.Values(tt.args.params).Encode()
			}
			req, _ := http.NewRequest(tt.args.method, p, nil)
			score := tt.fields.rule.Match(req, tt.args.body)
			require.Equal(t, tt.want.value, score)
		})
	}
}
