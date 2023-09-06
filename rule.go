package stubble

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const Wildcard = ".*"

type Response struct {
	// Response status code
	StatusCode int `json:"statusCode"`
	// Response body string. Only one body field can be used.
	BodyString string `json:"bodyString"`
	// Response body JSON. Only one body field can be used
	BodyJSON any `json:"bodyJson"`
	// Response headers
	Headers map[string][]string `json:"headers"`
}

type Rule struct {
	// The name of the rule.
	Name string `json:"name"`
	// A description, no functional impact.
	Description string `json:"description"`
	// HTTP method to match.
	Method string `json:"method"`
	// Path to match.
	Path string `json:"path"`
	// Path regex to match.
	PathRegex string `json:"pathRegex"`
	pathRegex *regexp.Regexp
	// Request query parameters. An entry with a zero-length value list will
	// only check for presence of the key.
	Params map[string][]string `json:"params"`
	// TODO(hvl): matching
	// Request headers. An entry with a zero-length value list will only check
	// for presence of the key.
	Headers map[string][]string `json:"headers"`
	// Request body to match as a string.
	BodyString string `json:"bodyString"`
	// Regex of body to match as a string.
	BodyStringRegex string `json:"bodyStringRegex"`
	// The JSON body to match. Field values can be a Wildcard.
	BodyJSON        any `json:"body"`
	bodyStringRegex *regexp.Regexp
	// The response to return.
	Response Response `json:"response"`
}

func (ru Rule) Match(r *http.Request, body []byte) int {
	score := 0
	score += matchMethod(ru, r)
	score += matchPath(ru, r)
	score += matchPathRegex(ru, r)
	score += matchParams(ru, r)
	score += matchHeaders(ru, r)
	score += matchBodyStringRegex(ru, body)
	score += matchBodyJSON(ru, body)
	return score
}

func matchMethod(ru Rule, r *http.Request) int {
	if ru.Method != "" {
		if ru.Method == r.Method {
			return 1
		}
		return -1000
	}
	return 0
}

func matchPath(ru Rule, r *http.Request) int {
	if ru.Path != "" {
		p, _, _ := strings.Cut(r.URL.Path, "?")
		if ru.Path == p {
			return 1
		}
		return -1000
	}
	return 0
}

func matchPathRegex(ru Rule, r *http.Request) int {
	if ru.pathRegex != nil {
		p, _, _ := strings.Cut(r.URL.Path, "?")
		if ru.pathRegex.MatchString(p) {
			return 1
		}
		return -1000
	}
	return 0
}

func matchParams(ru Rule, r *http.Request) int {
	if len(ru.Params) == 0 {
		return 0
	}
	score := 0
	q := r.URL.Query()
	for k, want := range ru.Params {
		got, ok := q[k]
		if !ok {
			return -1000
		}
		if len(want) == 0 {
			score += 1
			continue
		}
		if len(got) < len(want) {
			return -1000
		}
		x, ok := scoreStringSlice(want, got)
		if !ok {
			return -1000
		}
		score += x
	}
	return score
}

func matchHeaders(ru Rule, r *http.Request) int {
	if len(ru.Headers) == 0 {
		return 0
	}
	score := 0
	q := r.Header
	for k, want := range ru.Headers {
		got, ok := q[textproto.CanonicalMIMEHeaderKey(k)]
		if !ok {
			return -1000
		}
		if len(want) == 0 {
			score += 1
			continue
		}
		if len(got) < len(want) {
			return -1000
		}
		x, ok := scoreStringSlice(want, got)
		if !ok {
			return -1000
		}
		score += x
	}
	return score
}

func scoreStringSlice(want, got []string) (int, bool) {
	score := 0
	sort.Strings(got)
	i, j := 0, 0
	for i < len(want) && j < len(got) {
		switch {
		case want[i] == got[j]:
			i += 1
			j += 1
			score += 1
		case want[i] > got[j]:
			j += 1
		case want[i] < got[j]:
			return -1000, false
		}
	}
	return score, true
}

func matchBodyStringRegex(ru Rule, body []byte) int {
	if ru.bodyStringRegex != nil && ru.bodyStringRegex.Match(body) {
		return 1
	}
	return 0
}

func matchBodyJSON(ru Rule, body []byte) int {
	if ru.BodyJSON == nil {
		return 0
	}
	var v any
	err := json.Unmarshal(body, &v)
	if err != nil {
		return 0
	}
	if reflect.DeepEqual(ru.BodyJSON, v) {
		return 1
	}
	if arr, ok := v.([]any); ok {
		bd, ok := ru.BodyJSON.([]any)
		if !ok {
			return 0
		}
		if matchJSONArray(bd, arr) {
			return 1
		}
	}
	if obj, ok := v.(map[string]any); ok {
		bd, ok := ru.BodyJSON.(map[string]any)
		if !ok {
			return 0
		}
		if matchJSONObject(bd, obj) {
			return 1
		}
	}
	return 0
}

func matchJSONArray(bd []any, arr []any) bool {
	if len(bd) != len(arr) {
		return false
	}
	for i := range bd {
		if bd[i] == Wildcard {
			continue
		}
		switch x := bd[i].(type) {
		case []any:
			v2, ok := arr[i].([]any)
			if !ok {
				return false
			}
			if !matchJSONArray(x, v2) {
				return false
			}
		case map[string]any:
			x2, ok := arr[i].(map[string]any)
			if !ok {
				return false
			}
			if !matchJSONObject(x, x2) {
				return false
			}
		default:
			if !reflect.DeepEqual(x, arr[i]) {
				return false
			}
		}
	}
	return true
}

func matchJSONObject(bd map[string]any, obj map[string]any) bool {
	if len(bd) != len(obj) {
		return false
	}
	for k, v := range bd {
		if v == Wildcard {
			continue
		}
		switch x := v.(type) {
		case []any:
			v2, ok := obj[k]
			if !ok {
				return false
			}
			x2, ok := v2.([]any)
			if !ok {
				return false
			}
			if !matchJSONArray(x, x2) {
				return false
			}
		case map[string]any:
			v2, ok := obj[k]
			if !ok {
				return false
			}
			x2, ok := v2.(map[string]any)
			if !ok {
				return false
			}
			if !matchJSONObject(x, x2) {
				return false
			}
		default:
			if !reflect.DeepEqual(v, obj[k]) {
				return false
			}
		}
	}
	return true
}

func (ru Rule) EqualMatch(other Rule) bool {
	ru.Response = Response{}
	other.Response = Response{}
	return reflect.DeepEqual(ru, other)
}

func InitRule(r Rule) (Rule, error) {
	if r.PathRegex != "" {
		var err error
		r.pathRegex, err = regexp.Compile(r.PathRegex)
		if err != nil {
			return Rule{}, fmt.Errorf("error compiling path regex: %w", err)
		}
	}
	if r.BodyStringRegex != "" {
		var err error
		r.bodyStringRegex, err = regexp.Compile(r.BodyStringRegex)
		if err != nil {
			return Rule{}, fmt.Errorf("error compiling body regex: %w", err)
		}
	}
	for k, vs := range r.Headers {
		sort.Strings(vs)
		r.Headers[textproto.CanonicalMIMEHeaderKey(k)] = vs
	}
	return r, nil
}

func RuleFromRequest(r *http.Request) Rule {
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		getLogger().Infof("could not read request body")
	}
	defer func() { _ = r.Body.Close() }()
	ru := Rule{
		Method:     r.Method,
		Path:       r.URL.Path,
		Params:     r.URL.Query(),
		Headers:    r.Header,
		BodyString: string(bs),
	}
	return ru
}
