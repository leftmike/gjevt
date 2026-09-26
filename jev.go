package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, httpClient: http.DefaultClient}
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
		return nil, fmt.Errorf("jev: %s: %s", hresp.Status, b)
	}

	var resp DecisionResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
