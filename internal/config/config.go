package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Ticket      string `yaml:"ticket"`
	Description string `yaml:"description"`
}

type ProjectConfig struct {
	Projects map[string]Project `yaml:"projects"`
}

type JiraCredentials struct {
	CloudID  string
	BaseURL  string
	Email    string
	APIToken string
}

func LoadProjectConfig(filepath string) (*ProjectConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading project config: %w", err)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing project config: %w", err)
	}

	if err := validateProjectConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid project config: %w", err)
	}

	return &config, nil
}

func LoadCredentialsFromEnv() (*JiraCredentials, error) {
	creds := &JiraCredentials{
		CloudID:  os.Getenv("JIRA_CLOUD_ID"),
		BaseURL:  os.Getenv("JIRA_BASE_URL"),
		Email:    os.Getenv("JIRA_EMAIL"),
		APIToken: os.Getenv("JIRA_TOKEN"),
	}

	if creds.BaseURL == "" {
		creds.BaseURL = "https://api.atlassian.com"
	}

	if err := validateCredentials(creds); err != nil {
		return nil, fmt.Errorf("invalid environment credentials: %w", err)
	}

	return creds, nil
}

func validateProjectConfig(config *ProjectConfig) error {
	if len(config.Projects) == 0 {
		return fmt.Errorf("no projects defined")
	}

	seenTickets := make(map[string]string)
	for code, project := range config.Projects {
		if project.Ticket == "" {
			return fmt.Errorf("project %s has no ticket", code)
		}
		if prev, exists := seenTickets[project.Ticket]; exists {
			return fmt.Errorf("ticket %s used by both %s and %s", project.Ticket, prev, code)
		}
		seenTickets[project.Ticket] = code
	}

	return nil
}

func validateCredentials(creds *JiraCredentials) error {
	if creds.CloudID == "" {
		return fmt.Errorf("JIRA_CLOUD_ID environment variable required")
	}
	if creds.BaseURL == "" {
		return fmt.Errorf("JIRA_BASE_URL environment variable required")
	}
	if creds.Email == "" {
		return fmt.Errorf("JIRA_EMAIL environment variable required")
	}
	if creds.APIToken == "" {
		return fmt.Errorf("JIRA_TOKEN environment variable required")
	}
	return nil
}
