package loadtest

import "github.com/namnv2496/mocktool/internal/domain"

// Account represents a user account loaded from Excel
type Account struct {
	Username string
	Password string
	Extra    map[string]string // Additional columns from Excel
}

// Scenario wraps domain.LoadTestScenario for runtime use
type Scenario struct {
	Name        string
	Concurrency int
	Steps       []Step
	Accounts    string
}

// Step wraps domain.LoadTestStep for runtime use
type Step struct {
	Name             string
	Method           string
	Path             string
	Headers          map[string]string
	Body             string
	SaveVariables    map[string]string // Map of account_id -> map[variable_name]value
	ExpectStatus     int
	WaitAfterSeconds int    // Wait duration in seconds after step execution
	RetryForSeconds  int    // Retry duration in seconds if request fails
	MaxRetryTimes    int    // Maximum number of retry attempts (0 = unlimited within time limit)
	Condition        string // Condition to execute this step (e.g., "{{need_payment}} == true")
}

// FromDomain converts domain.LoadTestScenario to loadtest.Scenario
func FromDomain(s *domain.LoadTestScenario) *Scenario {
	steps := make([]Step, len(s.Steps))
	for i, ds := range s.Steps {
		steps[i] = Step{
			Name:             ds.Name,
			Method:           ds.Method,
			Path:             ds.Path,
			Headers:          ds.Headers,
			Body:             ds.Body,
			SaveVariables:    ds.SaveVariables,
			ExpectStatus:     ds.ExpectStatus,
			WaitAfterSeconds: ds.WaitAfterSeconds,
			RetryForSeconds:  ds.RetryForSeconds,
			MaxRetryTimes:    ds.MaxRetryTimes,
			Condition:        ds.Condition,
		}
	}
	return &Scenario{
		Name:  s.Name,
		Steps: steps,
	}
}

// FromDomainWithAccounts converts domain.LoadTestScenario to loadtest.Scenario and extracts accounts
// func FromDomainWithAccounts(s *domain.LoadTestScenario) (*Scenario, []Account) {
// 	scenario := FromDomain(s)

// 	accounts := make([]Account, len(s.Accounts))
// 	for i, da := range s.Accounts {
// 		accounts[i] = Account{
// 			Username: da.Username,
// 			Password: da.Password,
// 			Extra:    da.Extra,
// 		}
// 	}

// 	return scenario, accounts
// }

// ToDomain converts loadtest.Scenario to domain.LoadTestScenario
func ToDomain(s *Scenario, description string) *domain.LoadTestScenario {
	steps := make([]domain.LoadTestStep, len(s.Steps))
	for i, st := range s.Steps {
		steps[i] = domain.LoadTestStep{
			Name:             st.Name,
			Method:           st.Method,
			Path:             st.Path,
			Headers:          st.Headers,
			Body:             st.Body,
			SaveVariables:    st.SaveVariables,
			ExpectStatus:     st.ExpectStatus,
			WaitAfterSeconds: st.WaitAfterSeconds,
			RetryForSeconds:  st.RetryForSeconds,
			MaxRetryTimes:    st.MaxRetryTimes,
			Condition:        st.Condition,
		}
	}
	return &domain.LoadTestScenario{
		Name:        s.Name,
		Description: description,
		Steps:       steps,
		IsActive:    true,
	}
}

// ToDomainWithAccounts converts loadtest.Scenario and accounts to domain.LoadTestScenario
// func ToDomainWithAccounts(s *Scenario, accounts []Account, description string) *domain.LoadTestScenario {
// 	domainScenario := ToDomain(s, description)

// 	domainAccounts := make([]domain.LoadTestAccount, len(accounts))
// 	for i, acc := range accounts {
// 		domainAccounts[i] = domain.LoadTestAccount{
// 			Username: acc.Username,
// 			Password: acc.Password,
// 			Extra:    acc.Extra,
// 		}
// 	}

// 	domainScenario.Accounts = domainAccounts
// 	return domainScenario
// }

// StepResult represents the result of executing a single step
type StepResult struct {
	StepName     string
	StatusCode   int
	Duration     int64 // milliseconds
	Success      bool
	ErrorMessage string
	MetaData     map[string]string
}

// AccountResult represents the result of running all steps for an account
type AccountResult struct {
	Username    string
	StepResults []StepResult
	TotalTime   int64 // milliseconds
	Success     bool
}

// LoadTestResult represents the aggregated result of a load test
type LoadTestResult struct {
	ScenarioName   string
	TotalAccounts  int
	SuccessCount   int
	FailureCount   int
	TotalDuration  int64 // milliseconds
	AvgDuration    int64 // milliseconds
	AccountResults []AccountResult
}
