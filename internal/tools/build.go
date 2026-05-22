package tools

// BuildAll constructs the registry with every supported tool. It is the single
// entry point used by both cmd/mcpserver and cmd/slackbot — adding or removing
// a tool means editing exactly this slice.
func BuildAll(d Deps) *Registry {
	return NewRegistry(
		// Read
		listFeatures(d),
		listScenarios(d),
		searchScenarios(d),
		getActiveScenario(d),
		listAPIs(d),
		searchMocks(d),

		// Write
		createFeature(d),
		updateFeature(d),
		enableFeature(d),
		createScenario(d),
		updateScenario(d),
		createMockAPI(d),
		updateMockAPI(d),
		resetMockAPICounter(d),
		activateScenario(d),

		// Destructive (Slack-side confirmation required)
		disableFeature(d),
		setScenarioInactive(d),
		deactivateScenario(d),
		deleteMockAPI(d),
		deleteScenario(d),
		deleteFeature(d),
	)
}
