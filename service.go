package stuble

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Logger interface {
	Infof(format string, a ...any)
	Warnf(format string, a ...any)
	Errorf(format string, a ...any)
	Fatalf(format string, a ...any)
}

type Persistence interface {
	ListRules(context.Context) ([]Rule, error)
	SaveRule(context.Context, Rule) error
}

type Handler struct {
	rules Persistence
}

var NotFound = Response{
	StatusCode: http.StatusNotFound,
	BodyString: "no response for request",
}

func (h *Handler) GetResponse(r *http.Request) (Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rs, err := h.rules.ListRules(ctx)
	if err != nil {
		return Response{}, err
	}
	var body []byte
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return Response{}, fmt.Errorf("error reading request body: %w", err)
		}
	}
	best := NotFound
	score := 0
	for _, ru := range rs {
		s := ru.Match(r, body)
		if s > score {
			best = ru.Resp
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

const XStubleSetRule = "X-Stuble-Set-Rule"

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ok, _ := strconv.ParseBool(r.Header.Get(XStubleSetRule)); ok {
		if err := h.AddRule(r); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}
	resp, err := h.GetResponse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error: " + err.Error()))
		return
	}
	w.WriteHeader(resp.StatusCode)
	if resp.BodyString != "" {
		_, _ = w.Write([]byte(resp.BodyString))
	}
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
