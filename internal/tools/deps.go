package tools

import (
	"github.com/namnv2496/mocktool/internal/repository"
)

// Deps is the bundle of dependencies the tool handlers need. It is constructed
// at process start (fx in cmd/mcpserver and cmd/slackbot) and threaded through
// to every handler closure.
type Deps struct {
	Feature         repository.IFeatureRepository
	Scenario        repository.IScenarioRepository
	AccountScenario repository.IAccountScenarioRepository
	MockAPI         repository.IMockAPIRepository
	Cache           repository.ICache
}
