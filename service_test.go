package stuble_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/HayyoVanLoon/stuble"
)

func TestHandler_GetResponse(t *testing.T) {
	get := stuble.Rule{
		Method:   http.MethodGet,
		Response: stuble.Response{StatusCode: http.StatusOK, BodyString: "get"},
	}
	getFoo := stuble.Rule{
		Method:   http.MethodGet,
		Path:     "/foo",
		Response: stuble.Response{StatusCode: http.StatusOK, BodyString: "getFoo"},
	}

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
			want{getFoo.Response, require.NoError},
		},
		{
			"more precise",
			fields{[]stuble.Rule{get, getFoo}},
			args{http.MethodGet, "/foo", nil},
			want{getFoo.Response, require.NoError},
		},
		{
			"fail on one",
			fields{[]stuble.Rule{get, getFoo}},
			args{http.MethodGet, "/bar", nil},
			want{get.Response, require.NoError},
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
