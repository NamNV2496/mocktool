package usecase

import (
	"context"

	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
)

type ITrie interface {
	Insert(request entity.APIRequest) error
	Remove(request entity.APIRequest)
	Search(request entity.APIRequest) *entity.APIResponse
}

type TrieNode struct {
	children  map[string]*TrieNode // feature => scenario => path
	hashInput string
	regexPath string
	output    any
}

func newTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[string]*TrieNode),
	}
}

type Trie struct {
	root        *TrieNode
	MockAPIRepo repository.IMockAPIRepository
}

func NewTrie(
	MockAPIRepo repository.IMockAPIRepository,
) *Trie {
	root := newTrieNode()
	activeApis, _ := MockAPIRepo.ListActiveAPIs(context.Background())
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
		newAPINode.hashInput = api.HashValue
		newAPINode.regexPath = api.RegexPath
		newAPINode.output = api.Output

		node.children[api.Path] = newAPINode
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
	newAPINode.regexPath = request.RegexPath
	newAPINode.output = request.Output

	node.children[request.Path] = newAPINode
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
	node.children[request.Path] = nil
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
	if childNode, ok := node.children[request.Path]; ok {
		return &entity.APIResponse{
			Output: childNode.output,
		}
	}
	// try with regex path

	return nil
}
