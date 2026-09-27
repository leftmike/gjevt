package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var ErrTooLong = errors.New("state is too long")

type Question struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type DecisionRequest struct {
	Model     string              `json:"model,omitempty"`
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
}

type Answer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Legend        map[string]string  `json:"legend,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
}

type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}

type DecisionResponse struct {
	ID       string            `json:"id"`
	Model    string            `json:"model"`
	Provider string            `json:"provider"`
	Answers  map[string]Answer `json:"answers"`
	Usage    Usage             `json:"usage"`
}

const (
	DefaultBaseURL = "https://openrouter.ai/api"
	DefaultModel   = "~typesafe/jev-latest"
)

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

func NewClient(cfg Config, opts ...Option) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	c := &Client{cfg: cfg, httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Decide(ctx context.Context, req DecisionRequest) (*DecisionResponse, error) {
	if req.Model == "" {
		req.Model = c.cfg.Model
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/alpha/decisions",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	hreq.Header.Set("Content-Type", "application/json")

	hresp, err := c.httpClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer hresp.Body.Close()

	b, err := io.ReadAll(hresp.Body)
	if err != nil {
		return nil, err
	}
	if hresp.StatusCode != http.StatusOK {
		if bytes.Contains(b, []byte("max_tokens_exceeded")) {
			return nil, ErrTooLong
		}
		return nil, fmt.Errorf("jev: %s: %s", hresp.Status, b)
	}

	var resp DecisionResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DecideTruncated(ctx context.Context, req DecisionRequest) (*DecisionResponse,
	bool, error) {

	resp, err := c.Decide(ctx, req)
	if !errors.Is(err, ErrTooLong) {
		return resp, false, err
	}

	req.State, err = copyState(req.State)
	if err != nil {
		return nil, false, err
	}
	truncated := false
	for {
		var shrunk bool
		req.State, shrunk = truncateState(req.State,
			min(stateSize(req.State)*3/4, maxStateSize))
		if !shrunk {
			return nil, truncated, ErrTooLong
		}
		truncated = true

		resp, err = c.Decide(ctx, req)
		if !errors.Is(err, ErrTooLong) {
			return resp, truncated, err
		}
	}
}
