package treeops

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type traverser struct {
	prefs NavigationPrefs
}

type LeafTraverser interface {
	Traverse(matchingNode *CandidateNode, pathNode *PathElement) ([]*CandidateNode, error)
}

func NewLeafTraverser(navigationPrefs NavigationPrefs) LeafTraverser {
	return &traverser{navigationPrefs}
}

func (t *traverser) keyMatches(key *yaml.Node, pathNode *PathElement) bool {
	return pathNode.Value == "[]" || Match(key.Value, pathNode.StringValue)
}

func (t *traverser) traverseMap(candidate *CandidateNode, pathNode *PathElement) ([]*CandidateNode, error) {
	// value.Content is a concatenated array of key, value,
	// so keys are in the even indexes, values in odd.
	// merge aliases are defined first, but we only want to traverse them
	// if we don't find a match directly on this node first.
	//TODO ALIASES, auto creation?

	var newMatches = make([]*CandidateNode, 0)

	node := candidate.Node

	var contents = node.Content
	for index := 0; index < len(contents); index = index + 2 {
		key := contents[index]
		value := contents[index+1]

		log.Debug("checking %v (%v)", key.Value, key.Tag)
		if t.keyMatches(key, pathNode) {
			log.Debug("MATCHED")
			newMatches = append(newMatches, &CandidateNode{
				Node:     value,
				Path:     append(candidate.Path, key.Value),
				Document: candidate.Document,
			})
		}
	}
	if len(newMatches) == 0 {
		//no matches, create one automagically
		valueNode := &yaml.Node{}
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: pathNode.StringValue}, valueNode)
		newMatches = append(newMatches, &CandidateNode{
			Node:     valueNode,
			Path:     append(candidate.Path, pathNode.StringValue),
			Document: candidate.Document,
		})

	}

	return newMatches, nil
}

func (t *traverser) traverseArray(candidate *CandidateNode, pathNode *PathElement) ([]*CandidateNode, error) {
	log.Debug("pathNode Value %v", pathNode.Value)
	if pathNode.Value == "[]" {

		var contents = candidate.Node.Content
		var newMatches = make([]*CandidateNode, len(contents))

		for index := 0; index < len(contents); index = index + 1 {
			newMatches[index] = &CandidateNode{
				Document: candidate.Document,
				Path:     append(candidate.Path, index),
				Node:     contents[index],
			}
		}
		return newMatches, nil

	}

	index := pathNode.Value.(int64)
	indexToUse := index
	contentLength := int64(len(candidate.Node.Content))
	for contentLength <= index {
		candidate.Node.Content = append(candidate.Node.Content, &yaml.Node{})
		contentLength = int64(len(candidate.Node.Content))
	}

	if indexToUse < 0 {
		indexToUse = contentLength + indexToUse
	}

	if indexToUse < 0 {
		return nil, fmt.Errorf("Index [%v] out of range, array size is %v", index, contentLength)
	}

	return []*CandidateNode{&CandidateNode{
		Node:     candidate.Node.Content[indexToUse],
		Document: candidate.Document,
		Path:     append(candidate.Path, index),
	}}, nil

}

func (t *traverser) Traverse(matchingNode *CandidateNode, pathNode *PathElement) ([]*CandidateNode, error) {
	log.Debug("Traversing %v", NodeToString(matchingNode))
	value := matchingNode.Node

	if value.Kind == 0 {
		log.Debugf("Guessing kind")
		// we must ahve added this automatically, lets guess what it should be now
		switch pathNode.Value.(type) {
		case int, int64:
			log.Debugf("probably an array")
			value.Kind = yaml.SequenceNode
		default:
			log.Debugf("probabel a map")
			value.Kind = yaml.MappingNode
		}
	}

	switch value.Kind {
	case yaml.MappingNode:
		log.Debug("its a map with %v entries", len(value.Content)/2)
		return t.traverseMap(matchingNode, pathNode)

	case yaml.SequenceNode:
		log.Debug("its a sequence of %v things!", len(value.Content))
		return t.traverseArray(matchingNode, pathNode)
	// 	default:

	// 		if head == "+" {
	// 			return n.appendArray(value, head, tail, pathStack)
	// 		} else if len(value.Content) == 0 && head == "**" {
	// 			return n.navigationStrategy.Visit(nodeContext)
	// 		}
	// 		return n.splatArray(value, head, tail, pathStack)
	// 	}
	// case yaml.AliasNode:
	// 	log.Debug("its an alias!")
	// 	DebugNode(value.Alias)
	// 	if n.navigationStrategy.FollowAlias(nodeContext) {
	// 		log.Debug("following the alias")
	// 		return n.recurse(value.Alias, head, tail, pathStack)
	// 	}
	// 	return nil
	case yaml.DocumentNode:
		log.Debug("digging into doc node")
		return t.Traverse(&CandidateNode{
			Node:     matchingNode.Node.Content[0],
			Document: matchingNode.Document}, pathNode)
	default:
		return nil, nil
	}
}