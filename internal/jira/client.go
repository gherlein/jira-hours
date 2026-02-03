package jira

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gherlein/jira-hours/internal/config"
)

type Client struct {
	credentials *config.JiraCredentials
	httpClient  *http.Client
}

type WorklogRequest struct {
	TimeSpent string `json:"timeSpent"`
	Started   string `json:"started"`
}

type WorklogResponse struct {
	ID             string `json:"id"`
	IssueID        string `json:"issueId"`
	TimeSpent      string `json:"timeSpent"`
	TimeSpentSeconds int  `json:"timeSpentSeconds"`
}

type ErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

type Worklog struct {
	ID               string `json:"id"`
	IssueID          string `json:"issueId"`
	Author           Author `json:"author"`
	Started          string `json:"started"`
	TimeSpent        string `json:"timeSpent"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
}

type Author struct {
	AccountID    string `json:"accountId"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
}

type WorklogsResponse struct {
	Worklogs   []Worklog `json:"worklogs"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
}

func NewClient(creds *config.JiraCredentials) *Client {
	return &Client{
		credentials: creds,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) AddWorklog(issueKey string, hours int, startDate time.Time) error {
	if hours == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/ex/jira/%s/rest/api/3/issue/%s/worklog",
		c.credentials.BaseURL,
		c.credentials.CloudID,
		issueKey,
	)

	timeSpent := fmt.Sprintf("%dh", hours)
	started := startDate.Format("2006-01-02T15:04:05.000-0700")

	worklog := WorklogRequest{
		TimeSpent: timeSpent,
		Started:   started,
	}

	jsonData, err := json.Marshal(worklog)
	if err != nil {
		return fmt.Errorf("marshaling worklog: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	authHeader := c.getAuthHeader()
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			if len(errResp.ErrorMessages) > 0 {
				return fmt.Errorf("jira api error (status %d): %s", resp.StatusCode, errResp.ErrorMessages[0])
			}
			for field, msg := range errResp.Errors {
				return fmt.Errorf("jira api error (status %d): %s: %s", resp.StatusCode, field, msg)
			}
		}
		return fmt.Errorf("jira api error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) TestConnection() error {
	url := fmt.Sprintf("%s/ex/jira/%s/rest/api/3/myself",
		c.credentials.BaseURL,
		c.credentials.CloudID,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating test request: %w", err)
	}

	authHeader := c.getAuthHeader()
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("testing connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connection test failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) GetWorklogs(issueKey string) ([]Worklog, error) {
	url := fmt.Sprintf("%s/ex/jira/%s/rest/api/3/issue/%s/worklog",
		c.credentials.BaseURL,
		c.credentials.CloudID,
		issueKey,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	authHeader := c.getAuthHeader()
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get worklogs failed (status %d): %s", resp.StatusCode, string(body))
	}

	var worklogsResp WorklogsResponse
	if err := json.Unmarshal(body, &worklogsResp); err != nil {
		return nil, fmt.Errorf("parsing worklogs response: %w", err)
	}

	return worklogsResp.Worklogs, nil
}

func (c *Client) DeleteWorklog(issueKey, worklogID string) error {
	url := fmt.Sprintf("%s/ex/jira/%s/rest/api/3/issue/%s/worklog/%s",
		c.credentials.BaseURL,
		c.credentials.CloudID,
		issueKey,
		worklogID,
	)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	authHeader := c.getAuthHeader()
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete worklog failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) FindMatchingWorklogs(issueKey string, targetDate time.Time, userEmail string) ([]Worklog, error) {
	worklogs, err := c.GetWorklogs(issueKey)
	if err != nil {
		return nil, err
	}

	matches := make([]Worklog, 0)
	for _, wl := range worklogs {
		if isSameDay(wl.Started, targetDate) && isMyWorklog(wl, userEmail) {
			matches = append(matches, wl)
		}
	}

	return matches, nil
}

func (c *Client) WorklogExists(issueKey string, targetDate time.Time, hours int, userEmail string) (bool, string, error) {
	worklogs, err := c.GetWorklogs(issueKey)
	if err != nil {
		return false, "", err
	}

	for _, wl := range worklogs {
		if isSameDay(wl.Started, targetDate) && isMyWorklog(wl, userEmail) {
			expectedHours := hours * 3600
			if wl.TimeSpentSeconds == expectedHours {
				return true, wl.ID, nil
			}
		}
	}

	return false, "", nil
}

func isSameDay(worklogStarted string, targetDate time.Time) bool {
	worklogDate, err := time.Parse(time.RFC3339, worklogStarted)
	if err != nil {
		worklogDate, err = time.Parse("2006-01-02T15:04:05.000-0700", worklogStarted)
		if err != nil {
			return false
		}
	}

	wYear, wMonth, wDay := worklogDate.Date()
	tYear, tMonth, tDay := targetDate.Date()

	return wYear == tYear && wMonth == tMonth && wDay == tDay
}

func isMyWorklog(worklog Worklog, myEmail string) bool {
	return worklog.Author.EmailAddress == myEmail
}

func (c *Client) getAuthHeader() string {
	auth := c.credentials.Email + ":" + c.credentials.APIToken
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return "Basic " + encoded
}

type MockClient struct {
	loggedEntries    []WorklogRequest
	existingWorklogs map[string][]Worklog
	nextWorklogID    int
}

func NewMockClient() *MockClient {
	return &MockClient{
		loggedEntries:    make([]WorklogRequest, 0),
		existingWorklogs: make(map[string][]Worklog),
		nextWorklogID:    1000,
	}
}

func (m *MockClient) AddWorklog(issueKey string, hours int, startDate time.Time) error {
	if hours == 0 {
		return nil
	}

	timeSpent := fmt.Sprintf("%dh", hours)
	started := startDate.Format("2006-01-02T15:04:05.000-0700")

	m.loggedEntries = append(m.loggedEntries, WorklogRequest{
		TimeSpent: timeSpent,
		Started:   started,
	})

	worklog := Worklog{
		ID:               fmt.Sprintf("%d", m.nextWorklogID),
		IssueID:          issueKey,
		Started:          started,
		TimeSpent:        timeSpent,
		TimeSpentSeconds: hours * 3600,
		Author: Author{
			EmailAddress: "test@example.com",
		},
	}
	m.nextWorklogID++

	m.existingWorklogs[issueKey] = append(m.existingWorklogs[issueKey], worklog)

	return nil
}

func (m *MockClient) GetWorklogs(issueKey string) ([]Worklog, error) {
	worklogs, exists := m.existingWorklogs[issueKey]
	if !exists {
		return []Worklog{}, nil
	}
	return worklogs, nil
}

func (m *MockClient) DeleteWorklog(issueKey, worklogID string) error {
	worklogs, exists := m.existingWorklogs[issueKey]
	if !exists {
		return fmt.Errorf("issue not found")
	}

	for i, wl := range worklogs {
		if wl.ID == worklogID {
			m.existingWorklogs[issueKey] = append(worklogs[:i], worklogs[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("worklog not found")
}

func (m *MockClient) FindMatchingWorklogs(issueKey string, targetDate time.Time, userEmail string) ([]Worklog, error) {
	worklogs, err := m.GetWorklogs(issueKey)
	if err != nil {
		return nil, err
	}

	matches := make([]Worklog, 0)
	for _, wl := range worklogs {
		if isSameDay(wl.Started, targetDate) && isMyWorklog(wl, userEmail) {
			matches = append(matches, wl)
		}
	}

	return matches, nil
}

func (m *MockClient) WorklogExists(issueKey string, targetDate time.Time, hours int, userEmail string) (bool, string, error) {
	worklogs, err := m.GetWorklogs(issueKey)
	if err != nil {
		return false, "", err
	}

	for _, wl := range worklogs {
		if isSameDay(wl.Started, targetDate) && isMyWorklog(wl, userEmail) {
			expectedHours := hours * 3600
			if wl.TimeSpentSeconds == expectedHours {
				return true, wl.ID, nil
			}
		}
	}

	return false, "", nil
}

func (m *MockClient) TestConnection() error {
	return nil
}

func (m *MockClient) GetLoggedEntries() []WorklogRequest {
	return m.loggedEntries
}

func (m *MockClient) SetUserEmail(email string) {
	for issueKey := range m.existingWorklogs {
		for i := range m.existingWorklogs[issueKey] {
			m.existingWorklogs[issueKey][i].Author.EmailAddress = email
		}
	}
}
