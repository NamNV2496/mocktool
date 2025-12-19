package entity

type APIRequest struct {
	FeatureName string `json:"feature_name"`
	Scenario    string `json:"scenario"`
	Path        string `json:"path"`
	RegexPath   string `json:"regex_path"` // regex_path
	HashInput   string `json:"hash_input"` // hashcode of input
	Output      any    `json:"output"`     // json response
}

type APIResponse struct {
	FeatureName string `json:"feature_name"`
	Scenario    string `json:"scenario"`
	Path        string `json:"path"`
	Output      any    `json:"output"` // json response
}
