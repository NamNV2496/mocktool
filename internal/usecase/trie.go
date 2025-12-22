package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
)

type ITrie interface {
	Insert(request entity.APIRequest) error
	Remove(request entity.APIRequest)
	RemoveScenario(featureName, scenarioName string)
	Search(request entity.APIRequest) *entity.APIResponse
}

type TrieNode struct {
	children  map[string]*TrieNode // feature => scenario => path
	hashInput bson.Raw
	method    string
	output    any
	Headers   map[string]string
}

func newTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[string]*TrieNode),
	}
}

// normalizePathWithQueryParams sorts query parameters alphabetically for consistent matching
func normalizePathWithQueryParams(path string) string {
	// Split path and query string
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return path // No query parameters
	}

	basePath := parts[0]
	queryString := parts[1]

	// Parse and re-encode to sort parameters
	queryValues, err := url.ParseQuery(queryString)
	if err != nil {
		return path // Return original if parsing fails
	}

	// Encode() automatically sorts keys alphabetically
	if len(queryValues) > 0 {
		return basePath + "?" + queryValues.Encode()
	}

	return basePath
}

type Trie struct {
	root         *TrieNode
	MockAPIRepo  repository.IMockAPIRepository
	ScenarioRepo repository.IScenarioRepository
}

func NewTrie(
	MockAPIRepo repository.IMockAPIRepository,
	ScenarioRepo repository.IScenarioRepository,
) *Trie {
	ctx := context.Background()
	root := newTrieNode()
	// Load all active APIs into the trie
	// The scenario filtering happens at request time via AccountScenario mapping
	activeApis, _ := MockAPIRepo.ListAllActiveAPIs(ctx)
	for _, api := range activeApis {
		node := root
		// Navigate to or create feature node
		if childNode, ok := node.children[api.FeatureName]; !ok {
			newFeatureNode := newTrieNode()
			node.children[api.FeatureName] = newFeatureNode
			node = newFeatureNode
		} else {
			node = childNode
		}
		// Navigate to or create scenario node
		if childNode, ok := node.children[api.ScenarioName]; !ok {
			newScenarioNode := newTrieNode()
			node.children[api.ScenarioName] = newScenarioNode
			node = newScenarioNode
		} else {
			node = childNode
		}
		// Create API node with all properties
		newAPINode := newTrieNode()
		newAPINode.hashInput = api.HashInput
		newAPINode.method = api.Method
		newAPINode.output = api.Output
		var headerData map[string]string
		err := bson.Unmarshal(api.Headers, &headerData)
		if err == nil {
			newAPINode.Headers = headerData
		}
		// Normalize path to ensure query parameters are sorted
		normalizedPath := normalizePathWithQueryParams(api.Path)
		key := normalizedPath + hashInputKey(api.HashInput)
		node.children[key] = newAPINode
	}
	return &Trie{
		root:        root,
		MockAPIRepo: MockAPIRepo,
	}
}

func (_self *Trie) Insert(request entity.APIRequest) error {
	node := _self.root
	if childNode, ok := node.children[request.FeatureName]; !ok {
		newFeatureNode := newTrieNode()
		node.children[request.FeatureName] = newFeatureNode
		node = newFeatureNode
	} else {
		node = childNode
	}
	if childNode, ok := node.children[request.Scenario]; !ok {
		newScenarioNode := newTrieNode()
		node.children[request.Scenario] = newScenarioNode
		node = newScenarioNode
	} else {
		node = childNode
	}
	newAPINode := newTrieNode()
	newAPINode.hashInput = request.HashInput
	newAPINode.method = request.Method
	newAPINode.output = request.Output
	newAPINode.Headers = request.Headers
	// Normalize path to ensure query parameters are sorted
	normalizedPath := normalizePathWithQueryParams(request.Path)
	key := normalizedPath + hashInputKey(request.HashInput)
	node.children[key] = newAPINode
	return nil
}

func (_self *Trie) Remove(request entity.APIRequest) {
	node := _self.root
	if childNode, ok := node.children[request.FeatureName]; !ok {
		return
	} else {
		node = childNode
	}

	if childNode, ok := node.children[request.Scenario]; !ok {
		return
	} else {
		node = childNode
	}
	// Normalize path to ensure query parameters are sorted
	normalizedPath := normalizePathWithQueryParams(request.Path)
	key := normalizedPath + hashInputKey(request.HashInput)
	delete(node.children, key)
}

func (_self *Trie) RemoveScenario(featureName, scenarioName string) {
	node := _self.root
	// Navigate to feature node
	featureNode, ok := node.children[featureName]
	if !ok {
		return // Feature doesn't exist
	}

	// Remove the entire scenario subtree
	delete(featureNode.children, scenarioName)
}

// formatBsonOrJSON tries to decode BSON or JSON and format as pretty JSON for display
func formatBsonOrJSON(data []byte) string {
	if len(data) == 0 {
		return "<empty>"
	}

	var doc map[string]any

	// Try BSON first
	if err := bson.Unmarshal(data, &doc); err != nil {
		// Try JSON
		if err := json.Unmarshal(data, &doc); err != nil {
			// Return raw string if both fail
			return string(data)
		}
	}

	// Marshal back to pretty JSON
	prettyJSON, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(prettyJSON)
}

// compareBsonRaw compares two bson.Raw values by treating them as JSON and comparing structure
func compareBsonRaw(a, b bson.Raw) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	// Format both inputs first for better debugging
	formattedA := formatBsonOrJSON(a)
	formattedB := formatBsonOrJSON(b)

	println("Comparing A:", formattedA)
	println("Comparing B:", formattedB)

	// Try to unmarshal as BSON first, then fall back to JSON
	var docA, docB map[string]any

	// Try BSON first for 'a', fall back to JSON
	if err := bson.Unmarshal(a, &docA); err != nil {
		// Try as JSON
		if err := json.Unmarshal(a, &docA); err != nil {
			println("Error unmarshaling 'a':", err.Error(), "Data:", string(a))
			return false
		}
	}

	// Try BSON first for 'b', fall back to JSON
	if err := bson.Unmarshal(b, &docB); err != nil {
		// Try as JSON
		if err := json.Unmarshal(b, &docB); err != nil {
			println("Error unmarshaling 'b':", err.Error(), "Data:", string(b))
			return false
		}
	}

	// Marshal with sorted keys for canonical comparison
	sortedA, err := json.Marshal(sortMapKeys(docA))
	if err != nil {
		println("Error marshaling sortedA:", err.Error())
		return false
	}
	sortedB, err := json.Marshal(sortMapKeys(docB))
	if err != nil {
		println("Error marshaling sortedB:", err.Error())
		return false
	}

	return bytes.Equal(sortedA, sortedB)
}

func hashInputKey(input bson.Raw) string {
	// Try to unmarshal as BSON first, then fall back to JSON
	var docA map[string]any

	// Try BSON first, fall back to JSON
	if err := bson.Unmarshal(input, &docA); err != nil {
		// Try as JSON
		if err := json.Unmarshal(input, &docA); err != nil {
			// Return empty hash if both fail
			hash := sha256.Sum256(input)
			return hex.EncodeToString(hash[:])
		}
	}

	// Marshal with sorted keys for canonical comparison
	sortedA, err := json.Marshal(sortMapKeys(docA))
	if err != nil {
		hash := sha256.Sum256(input)
		return hex.EncodeToString(hash[:])
	}

	// Generate SHA-256 hash and return as hex string
	hash := sha256.Sum256(sortedA)
	return "-" + hex.EncodeToString(hash[:])
}

// sortMapKeys recursively sorts map keys for canonical representation
func sortMapKeys(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		// Recursively sort nested maps
		if nestedMap, ok := v.(map[string]any); ok {
			v = sortMapKeys(nestedMap)
		}
		result[k] = v
	}
	return result
}

func (_self *Trie) Search(request entity.APIRequest) *entity.APIResponse {
	node := _self.root
	if childNode, ok := node.children[request.FeatureName]; !ok {
		return nil
	} else {
		node = childNode
	}
	if childNode, ok := node.children[request.Scenario]; !ok {
		return nil
	} else {
		node = childNode
	}
	// try with correct path
	// Normalize path to ensure query parameters are sorted
	normalizedPath := normalizePathWithQueryParams(request.Path)
	key := normalizedPath + hashInputKey(request.HashInput)
	if childNode, ok := node.children[key]; ok {
		matchMethod := childNode.method == request.Method
		matchInput := compareBsonRaw(childNode.hashInput, request.HashInput)
		if matchMethod && matchInput {
			return &entity.APIResponse{
				Output:  childNode.output,
				Headers: childNode.Headers,
			}
		}
	}
	// try with regex path

	return nil
}
