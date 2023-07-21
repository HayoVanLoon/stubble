package stuble

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const Wildcard = ".*"

type Response struct {
	StatusCode int    `json:"statusCode"`
	BodyString string `json:"bodyString"`
	BodyJSON   any    `json:"bodyJson"`
}

type Rule struct {
	Name      string `json:"name"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	PathRegex string `json:"pathRegex"`
	pathRegex *regexp.Regexp
	Params    map[string][]string `json:"params"`
	// TODO(hvl): matching
	Headers         map[string][]string `json:"headers"`
	BodyString      string              `json:"bodyString"`
	BodyStringRegex string              `json:"bodyStringRegex"`
	// map or slice
	BodyJSON        any `json:"body"`
	bodyStringRegex *regexp.Regexp
	Response        Response `json:"response"`
}

func (ru Rule) Match(r *http.Request, body []byte) int {
	score := 0
	score += matchMethod(ru, r)
	score += matchPath(ru, r)
	score += matchPathRegex(ru, r)
	score += matchParams(ru, r)
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
	q := r.URL.Query()
	for k, vs := range ru.Params {
		vs2, ok := q[k]
		if !ok {
			return -1000
		}
		ss := sort.StringSlice(vs2)
		for _, v := range vs {
			if ss.Search(v) < 0 {
				return -1000
			}
		}
	}
	return 1
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
