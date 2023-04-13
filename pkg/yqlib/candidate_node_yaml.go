package yqlib

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

func MapYamlStyle(original yaml.Style) Style {
	switch original {
	case yaml.TaggedStyle:
		return TaggedStyle
	case yaml.DoubleQuotedStyle:
		return DoubleQuotedStyle
	case yaml.SingleQuotedStyle:
		return SingleQuotedStyle
	case yaml.LiteralStyle:
		return LiteralStyle
	case yaml.FoldedStyle:
		return FoldedStyle
	case yaml.FlowStyle:
		return FlowStyle
	}
	return 0
}

func MapToYamlStyle(original Style) yaml.Style {
	switch original {
	case TaggedStyle:
		return yaml.TaggedStyle
	case DoubleQuotedStyle:
		return yaml.DoubleQuotedStyle
	case SingleQuotedStyle:
		return yaml.SingleQuotedStyle
	case LiteralStyle:
		return yaml.LiteralStyle
	case FoldedStyle:
		return yaml.FoldedStyle
	case FlowStyle:
		return yaml.FlowStyle
	}
	return 0
}

func (o *CandidateNode) copyFromYamlNode(node *yaml.Node) {
	o.Style = MapYamlStyle(node.Style)

	o.Tag = node.Tag
	o.Value = node.Value
	o.Anchor = node.Anchor

	// o.Alias = TODO - find Alias in our own structure
	// might need to be a post process thing

	o.HeadComment = node.HeadComment
	o.LineComment = node.LineComment
	o.FootComment = node.FootComment

	o.Line = node.Line
	o.Column = node.Column
}

func (o *CandidateNode) copyToYamlNode(node *yaml.Node) {
	node.Style = MapToYamlStyle(o.Style)

	node.Tag = o.Tag
	node.Value = o.Value
	node.Anchor = o.Anchor

	// node.Alias = TODO - find Alias in our own structure
	// might need to be a post process thing

	node.HeadComment = o.HeadComment
	node.LineComment = o.LineComment
	node.FootComment = o.FootComment

	node.Line = o.Line
	node.Column = o.Column
}

func (o *CandidateNode) decodeIntoChild(childNode *yaml.Node) (*CandidateNode, error) {
	newChild := o.CreateChild()

	// null yaml.Nodes to not end up calling UnmarshalYAML
	// so we call it explicitly
	if childNode.Tag == "!!null" {
		newChild.Kind = ScalarNode
		newChild.copyFromYamlNode(childNode)
		return newChild, nil
	}

	err := childNode.Decode(newChild)
	return newChild, err
}

func (o *CandidateNode) UnmarshalYAML(node *yaml.Node) error {
	log.Debugf("UnmarshalYAML %v", node.Tag)
	switch node.Kind {
	case yaml.DocumentNode:
		log.Debugf("UnmarshalYAML -  a document")
		o.Kind = DocumentNode
		o.copyFromYamlNode(node)
		if len(node.Content) == 0 {
			return nil
		}

		singleChild, err := o.decodeIntoChild(node.Content[0])

		if err != nil {
			return err
		}
		o.Content = []*CandidateNode{singleChild}
		return nil
	case yaml.AliasNode:
		log.Debug("UnmarshalYAML - alias from yaml: %v", o.Tag)
		o.Kind = AliasNode
		o.copyFromYamlNode(node)
		return nil
	case yaml.ScalarNode:
		log.Debugf("UnmarshalYAML -  a scalar")
		o.Kind = ScalarNode
		o.copyFromYamlNode(node)
		return nil
	case yaml.MappingNode:
		log.Debugf("UnmarshalYAML -  a mapping node")
		o.Kind = MappingNode
		o.copyFromYamlNode(node)
		o.Content = make([]*CandidateNode, len(node.Content))
		for i := 0; i < len(node.Content); i += 2 {

			keyNode, err := o.decodeIntoChild(node.Content[i])
			if err != nil {
				return err
			}

			keyNode.IsMapKey = true

			valueNode, err := o.decodeIntoChild(node.Content[i+1])
			if err != nil {
				return err
			}

			valueNode.Key = keyNode

			o.Content[i] = keyNode
			o.Content[i+1] = valueNode
		}
		return nil
	case yaml.SequenceNode:
		log.Debugf("UnmarshalYAML -  a sequence: %v", len(node.Content))
		o.Kind = SequenceNode
		o.copyFromYamlNode(node)
		o.Content = make([]*CandidateNode, len(node.Content))
		for i := 0; i < len(node.Content); i += 1 {
			keyNode := o.CreateChild()
			keyNode.IsMapKey = true // can't remember if we need this for sequences
			keyNode.Tag = "!!int"
			keyNode.Kind = ScalarNode
			keyNode.Value = fmt.Sprintf("%v", i)

			valueNode, err := o.decodeIntoChild(node.Content[i])
			if err != nil {
				return err
			}

			valueNode.Key = keyNode
			o.Content[i] = valueNode
		}
		return nil
	case 0:
		log.Debugf("UnmarshalYAML -  errr.. %v", node.Tag)
		// not sure when this happens
		o.copyFromYamlNode(node)
		return nil
	default:
		return fmt.Errorf("orderedMap: invalid yaml node")
	}
}

func (o *CandidateNode) MarshalYAML() (interface{}, error) {
	log.Debug("encoding to yaml: %v", o.Tag)
	switch o.Kind {
	case DocumentNode:
		target := &yaml.Node{Kind: yaml.DocumentNode}
		o.copyToYamlNode(target)

		singleChild := &yaml.Node{}
		err := singleChild.Encode(o.Content[0])
		if err != nil {
			return nil, err
		}
		target.Content = []*yaml.Node{singleChild}
		return target, nil
	case AliasNode:
		log.Debug("encoding alias to yaml: %v", o.Tag)
		target := &yaml.Node{Kind: yaml.AliasNode}
		o.copyToYamlNode(target)
		return target, nil
	case ScalarNode:
		log.Debug("encoding scalar: %v", o.Value)
		target := &yaml.Node{Kind: yaml.ScalarNode}
		o.copyToYamlNode(target)
		return target, nil
	case MappingNode, SequenceNode:
		targetKind := yaml.MappingNode
		if o.Kind == SequenceNode {
			targetKind = yaml.SequenceNode
		}
		target := &yaml.Node{Kind: targetKind}
		o.copyToYamlNode(target)
		target.Content = make([]*yaml.Node, len(o.Content))
		for i := 0; i < len(o.Content); i += 1 {
			child := &yaml.Node{}
			err := child.Encode(o.Content[i])
			if err != nil {
				return nil, err
			}
			log.Debug("child type %v", child.Tag)
			log.Debug("child is doc %v", child.Kind == yaml.DocumentNode)
			target.Content[i] = child
		}
		return target, nil
	}
	target := &yaml.Node{}
	o.copyToYamlNode(target)
	return target, nil
}