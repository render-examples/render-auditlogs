package render

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Actor struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	ID    string `json:"id"`
}

type AuditLog struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Event     string            `json:"event"`
	Status    string            `json:"status"`
	Actor     Actor             `json:"actor"`
	Metadata  map[string]string `json:"metadata"`
}

type AuditLogEntry struct {
	Cursor   string   `json:"cursor"`
	AuditLog AuditLog `json:"auditLog"`
}

type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) GetAuditLogs(endpoint string, cursor string, limit int) ([]AuditLogEntry, error) {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, err
	}

	q := url.Values{
		"direction": []string{"forward"},
		"limit":     []string{fmt.Sprintf("%d", limit)},
		"cursor":    []string{cursor},
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var auditLogs []AuditLogEntry
	err = json.Unmarshal(body, &auditLogs)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %w", err)
	}

	return auditLogs, nil
}
