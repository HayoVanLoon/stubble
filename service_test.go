package stubble_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/HayoVanLoon/stubble"
)

func TestHandler_GetResponse(t *testing.T) {
	get := stubble.Rule{
		Method:   http.MethodGet,
		Response: stubble.Response{StatusCode: http.StatusOK, BodyString: "get"},
	}
	getFoo := stubble.Rule{
		Method:   http.MethodGet,
		Path:     "/foo",
		Response: stubble.Response{StatusCode: http.StatusOK, BodyString: "getFoo"},
	}

	type fields struct {
		rules []stubble.Rule
	}
	type args struct {
		method string
		path   string
		body   []byte
	}
	type want struct {
		value stubble.Rule
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
			fields{[]stubble.Rule{getFoo}},
			args{http.MethodGet, "/foo", nil},
			want{getFoo, require.NoError},
		},
		{
			"more precise",
			fields{[]stubble.Rule{get, getFoo}},
			args{http.MethodGet, "/foo", nil},
			want{getFoo, require.NoError},
		},
		{
			"fail on one",
			fields{[]stubble.Rule{get, getFoo}},
			args{http.MethodGet, "/bar", nil},
			want{get, require.NoError},
		},
		{
			"not found",
			fields{[]stubble.Rule{getFoo}},
			args{http.MethodGet, "/moo", nil},
			want{stubble.NotFound, require.NoError},
		},
		{
			"no rules",
			fields{},
			args{http.MethodGet, "/foo", nil},
			want{stubble.NotFound, require.NoError},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := stubble.New(tt.fields.rules...)

			req, _ := http.NewRequest(tt.args.method, tt.args.path, bytes.NewReader(tt.args.body))
			actual, err := h.GetRule(req, tt.args.body)
			tt.want.err(t, err)
			require.Equal(t, tt.want.value, actual)
		})
	}
}
