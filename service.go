package stubble

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Logger interface {
	Debugf(format string, a ...any)
	Infof(format string, a ...any)
	Warnf(format string, a ...any)
	Errorf(format string, a ...any)
	Fatalf(format string, a ...any)
}

type Persistence interface {
	ListRules(context.Context) ([]Rule, error)
	SaveRule(context.Context, Rule) error
}

// A Handler handles stubbed calls.
type Handler struct {
	rules    Persistence
	requests []Rule
}

// NotFound is used when nothing matches. A 5xx is returned as a 404 would be
// easier to confuse with a normal prepared response.
var NotFound = Rule{
	Name: "no_match",
	Response: Response{
		StatusCode: http.StatusNotImplemented,
		BodyString: "no response for request",
	},
}

// GetRule retrieves the best matching response for the request.
func (h *Handler) GetRule(r *http.Request, body []byte) (Rule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rs, err := h.rules.ListRules(ctx)
	if err != nil {
		return Rule{}, err
	}
	best := NotFound
	score := 0
	for _, ru := range rs {
		s := ru.Match(r, body)
		if s > score {
			best = ru
			score = s
		}
	}
	getLogger().Infof("score: %d", score)
	return best, err
}

func (h *Handler) AddRule(r *http.Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var ru Rule
	err := json.NewDecoder(r.Body).Decode(&ru)
	if err != nil {
		return fmt.Errorf("error parsing rule: %w", err)
	}
	ru, err = InitRule(ru)
	if err != nil {
		return fmt.Errorf("error initialising rule: %w", err)
	}
	if err = h.rules.SaveRule(ctx, ru); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/stubble/") {
		h.handleStubbleRequests(w, r)
		getLogger().Infof("(stubble) %s %s", r.Method, r.URL.String())
		return
	}

	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("error: " + err.Error()))
			getLogger().Errorf("error reading request body: %v", err)
			return
		}
	}
	ru, err := h.GetRule(r, body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error: " + err.Error()))
		getLogger().Errorf("error fetching response: %v", err)
		return
	}

	w.WriteHeader(ru.Response.StatusCode)
	setHeaders(w, ru)
	respBody := buildResponseBody(ru.Response)
	if len(respBody) > 0 {
		_, _ = w.Write(respBody)
	}

	logResult(ru, r, body, respBody)
}

func (h *Handler) handleStubbleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Path != "/stubble/requests" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := struct {
			Requests []Rule `json:"requests"`
		}{Requests: h.requests}
		bs, _ := json.Marshal(resp)
		_, _ = w.Write(bs)
	case http.MethodPost:
		if r.URL.Path != "/stubble/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := h.AddRule(r); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func (h *Handler) storeRequest(r *http.Request) {
	const maxRequests = 40
	ru := RuleFromRequest(r)
	if len(h.requests) < maxRequests {
		h.requests = append(h.requests, ru)
		return
	}
	copy(h.requests, h.requests[1:])
	h.requests[len(h.requests)-1] = ru
}

func (h *Handler) Close() error {
	return nil
}

func FromFiles(files ...string) (*Handler, error) {
	var rss []Rule
	for _, f := range files {
		rs, err := readFile(f)
		if err != nil {
			return nil, err
		}
		rss = append(rss, rs...)
	}
	return New(rss...)
}

func New(rules ...Rule) (*Handler, error) {
	return &Handler{rules: inMemory{rules: rules}}, nil
}

func setHeaders(w http.ResponseWriter, ru Rule) {
	for k, vs := range ru.Response.Headers {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
}

func buildResponseBody(resp Response) []byte {
	if resp.BodyString != "" {
		return []byte(resp.BodyString)
	}
	if resp.BodyJSON != nil {
		bs, _ := json.Marshal(resp.BodyJSON)
		return bs
	}
	return nil
}

func logResult(ru Rule, r *http.Request, body, rBody []byte) {
	rep := fmt.Sprintf("%d|%s|%d", ru.Response.StatusCode, ru.Name, len(rBody))
	msg := fmt.Sprintf("(%s) %s %s", rep, r.Method, r.URL.String())
	if len(body) > 0 {
		msg += " <<< " + string(body)
	}
	getLogger().Infof(msg)
}

func readFile(f string) ([]Rule, error) {
	r, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	var rs []Rule
	for d.More() {
		var ru Rule
		err = d.Decode(&ru)
		if err != nil {
			return nil, err
		}
		ru, err = InitRule(ru)
		if err != nil {
			return nil, err
		}
		rs = append(rs, ru)
	}
	return rs, nil
}
