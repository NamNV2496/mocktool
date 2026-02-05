package loadtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/namnv2496/mocktool/internal/repository"
	"gopkg.in/yaml.v3"
)

// ScenarioLoader loads scenario configurations from YAML files or database
type ScenarioLoader struct {
	repo repository.ILoadTestScenarioRepository
}

// NewScenarioLoader creates a new ScenarioLoader
func NewScenarioLoader(repo repository.ILoadTestScenarioRepository) *ScenarioLoader {
	return &ScenarioLoader{repo: repo}
}

// LoadScenarioFromDB loads a scenario from the database by name
func (l *ScenarioLoader) LoadScenarioFromDB(ctx context.Context, scenarioName string) (*Scenario, error) {
	if l.repo == nil {
		return nil, fmt.Errorf("database repository not configured")
	}

	domainScenario, err := l.repo.GetByName(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario from database: %w", err)
	}

	return FromDomain(domainScenario), nil
}

// LoadScenarioWithAccountsFromDB loads a scenario with its embedded accounts from the database by name
func (l *ScenarioLoader) LoadScenarioWithAccountsFromDB(ctx context.Context, scenarioName string) (*Scenario, string, error) {
	if l.repo == nil {
		return nil, "", fmt.Errorf("database repository not configured")
	}

	domainScenario, err := l.repo.GetByName(ctx, scenarioName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get scenario from database: %w", err)
	}

	scenario := FromDomain(domainScenario)
	return scenario, domainScenario.Accounts, nil
}

// LoadScenarioFromFile loads a scenario from a YAML file
func (l *ScenarioLoader) LoadScenarioFromFile(scenarioDir, scenarioName string) (*Scenario, error) {
	yamlPath := filepath.Join(scenarioDir, scenarioName+".yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		// Try .yml extension
		yamlPath = filepath.Join(scenarioDir, scenarioName+".yml")
		data, err = os.ReadFile(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read scenario file: %w", err)
		}
	}

	// Parse YAML into a temporary struct that matches the file format
	var yamlScenario struct {
		Name        string `yaml:"name"`
		Concurrency int    `yaml:"concurrency"`
		Steps       []struct {
			Name          string            `yaml:"name"`
			Method        string            `yaml:"method"`
			Path          string            `yaml:"path"`
			Headers       map[string]string `yaml:"headers"`
			Body          string            `yaml:"body"`
			SaveVariables map[string]string `yaml:"save_variables"` // variable_name: json_path
			ExpectStatus  int               `yaml:"expect_status"`
		} `yaml:"steps"`
	}

	if err := yaml.Unmarshal(data, &yamlScenario); err != nil {
		return nil, fmt.Errorf("failed to parse scenario YAML: %w", err)
	}

	// Convert to Scenario
	scenario := &Scenario{
		Name:        yamlScenario.Name,
		Concurrency: yamlScenario.Concurrency,
		Steps:       make([]Step, len(yamlScenario.Steps)),
	}

	if scenario.Name == "" {
		scenario.Name = scenarioName
	}

	for i, s := range yamlScenario.Steps {
		scenario.Steps[i] = Step{
			Name:          s.Name,
			Method:        s.Method,
			Path:          s.Path,
			Headers:       s.Headers,
			Body:          s.Body,
			SaveVariables: s.SaveVariables,
			ExpectStatus:  s.ExpectStatus,
		}
	}

	return scenario, nil
}

// SaveScenarioToDB saves a scenario to the database
func (l *ScenarioLoader) SaveScenarioToDB(ctx context.Context, scenario *Scenario, description string) error {
	if l.repo == nil {
		return fmt.Errorf("database repository not configured")
	}

	domainScenario := ToDomain(scenario, description)
	return l.repo.Create(ctx, domainScenario)
}

// GetAccountsPath returns the expected Excel file path for a scenario
func (l *ScenarioLoader) GetAccountsPath(scenarioDir, scenarioName string) string {
	return filepath.Join(scenarioDir, scenarioName+".xlsx")
}
