package tools

// BuildAll constructs the registry with every supported tool. It is the single
// entry point used by both cmd/mcpserver and cmd/slackbot — adding or removing
// a tool means editing exactly this slice.
func BuildAll(d Deps) *Registry {
	return NewRegistry(
		// Read
		listFeatures(d),
		listScenarios(d),
		getActiveScenario(d),
		listAPIs(d),
		searchMocks(d),

		// Write
		createMockAPI(d),
		updateMockAPI(d),
		activateScenario(d),

		// Destructive (Slack-side confirmation required)
		deactivateScenario(d),
		deleteMockAPI(d),
		deleteScenario(d),
		deleteFeature(d),
	)
}
