package stuble

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
)

const Wildcard = "*"

type Response struct {
	StatusCode int    `json:"statusCode"`
	BodyString string `json:"body"`
}

type Rule struct {
	Method          string `json:"method"`
	Path            string `json:"path"`
	PathRegex       string `json:"pathRegex"`
	pathRegex       *regexp.Regexp
	Headers         map[string][]string `json:"headers"`
	BodyString      string              `json:"bodyString"`
	BodyStringRegex string              `json:"bodyStringRegex"`
	// map or slice
	BodyJSON        any `json:"body"`
	bodyStringRegex *regexp.Regexp
	Resp            Response `json:"resp"`
}

func (ru Rule) Match(r *http.Request, body []byte) int {
	score := 0
	score += matchMethod(ru, r)
	score += matchPath(ru, r)
	score += matchPathRegex(ru, r)
	score += matchBodyStringRegex(ru, body)
	score += matchBodyJSON(ru, body)
	return score
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

func matchMethod(ru Rule, r *http.Request) int {
	if ru.Method != "" && ru.Method != r.Method {
		return 1
	}
	return 0
}

func matchPath(ru Rule, r *http.Request) int {
	if ru.Path != "" && ru.Path == r.URL.Path {
		return 1
	}
	return 0
}

func matchPathRegex(ru Rule, r *http.Request) int {
	if ru.pathRegex != nil && ru.pathRegex.MatchString(r.URL.Path) {
		return 1
	}
	return 0
}

func matchBodyStringRegex(ru Rule, body []byte) int {
	if ru.bodyStringRegex != nil && ru.bodyStringRegex.Match(body) {
		return 1
	}
	return 0
}

func (ru Rule) EqualMatch(other Rule) bool {
	ru.Resp = Response{}
	other.Resp = Response{}
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
