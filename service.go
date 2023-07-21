package stuble

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
var NotFound = Response{
	StatusCode: http.StatusNotImplemented,
	BodyString: "no response for request",
}

// GetResponse retrieves the best matching response for the request.
func (h *Handler) GetResponse(r *http.Request, body []byte) (Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rs, err := h.rules.ListRules(ctx)
	if err != nil {
		return Response{}, err
	}
	best := NotFound
	score := 0
	for _, ru := range rs {
		s := ru.Match(r, body)
		if s > score {
			best = ru.Response
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
	if strings.HasPrefix(r.URL.Path, "/stuble/") {
		h.handleStubleRequests(w, r)
		getLogger().Infof("(stuble) %s %s", r.Method, r.URL.String())
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
	resp, err := h.GetResponse(r, body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error: " + err.Error()))
		getLogger().Errorf("error fetching response: %v", err)
		return
	}

	w.WriteHeader(resp.StatusCode)
	if resp.BodyString != "" {
		_, _ = w.Write([]byte(resp.BodyString))
	}

	logResult(resp, r, body)
}

func logResult(resp Response, r *http.Request, body []byte) {
	msg := fmt.Sprintf("(%d) %s %s", resp.StatusCode, r.Method, r.URL.String())
	if len(body) > 0 {
		msg += " <<< " + string(body)
	}
	getLogger().Infof(msg)
}

func (h *Handler) handleStubleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Path != "/stuble/requests" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := struct {
			Requests []Rule `json:"requests"`
		}{Requests: h.requests}
		bs, _ := json.Marshal(resp)
		_, _ = w.Write(bs)
	case http.MethodPost:
		if r.URL.Path != "/stuble/responses" {
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

func readFile(f string) ([]Rule, error) {
	r, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	d := json.NewDecoder(r)
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
