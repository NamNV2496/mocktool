package usecase

import (
	"fmt"

	"github.com/namnv2496/mocktool/internal/entity"
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

func NewTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[string]*TrieNode),
	}
}

type Trie struct {
	root *TrieNode
}

func NewTrie() *Trie {
	return &Trie{
		root: NewTrieNode(),
	}
}

func (_self *Trie) Insert(request entity.APIRequest) error {
	node := _self.root
	if childNode, ok := node.children[request.FeatureName]; !ok {
		return fmt.Errorf("feature not found")
	} else {
		node = childNode
	}
	if childNode, ok := node.children[request.Scenario]; !ok {
		return fmt.Errorf("scenario not found")
	} else {
		node = childNode
	}
	newAPINode := NewTrieNode()
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
