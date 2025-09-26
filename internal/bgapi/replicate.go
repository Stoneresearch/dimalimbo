package bgapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

type Client struct {
	HTTP  *http.Client
	Token string
	Model string
	Base  string
}

func NewClient(token, model string) *Client {
	return &Client{
		HTTP:  &http.Client{Timeout: 60 * time.Second},
		Token: token,
		Model: model,
		Base:  "https://api.replicate.com/v1",
	}
}

// Generate requests an image and returns the first output image URL.
func (c *Client) Generate(ctx context.Context, prompt string, width, height int) (string, error) {
	if c.Token == "" {
		return "", errors.New("missing replicate token")
	}
	model := c.Model
	if model == "" {
		model = "black-forest-labs/flux-1.1-pro"
	}
	body := map[string]any{
		"model": model,
		"input": map[string]any{
			"prompt":              prompt,
			"width":               width,
			"height":              height,
			"guidance":            3.5,
			"num_inference_steps": 28,
		},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/predictions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Token "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		x, _ := io.ReadAll(resp.Body)
		return "", errors.New(string(x))
	}
	var p struct {
		ID     string          `json:"id"`
		Status string          `json:"status"`
		Output json.RawMessage `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return "", err
	}

	// Poll until completed
	for i := 0; i < 40; i++ {
		if p.Status == "succeeded" || p.Status == "failed" || p.Status == "canceled" {
			break
		}
		time.Sleep(1500 * time.Millisecond)
		rq, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/predictions/"+p.ID, nil)
		rq.Header.Set("Authorization", "Token "+c.Token)
		rs, err := c.HTTP.Do(rq)
		if err != nil {
			return "", err
		}
		var pr struct {
			Status string          `json:"status"`
			Output json.RawMessage `json:"output"`
		}
		_ = json.NewDecoder(rs.Body).Decode(&pr)
		rs.Body.Close()
		p.Status = pr.Status
		p.Output = pr.Output
	}

	if p.Status != "succeeded" {
		return "", errors.New("replicate did not succeed: " + p.Status)
	}
	// Output is typically an array of URLs
	var urls []string
	_ = json.Unmarshal(p.Output, &urls)
	if len(urls) == 0 {
		return "", errors.New("no output images")
	}
	return urls[0], nil
}
