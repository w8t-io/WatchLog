package nodeInfo

import (
	"fmt"
)

// LogInfoNode node info
type LogInfoNode struct {
	Value    string
	Children map[string]*LogInfoNode
}

func NewLogInfoNode(value string) *LogInfoNode {
	return &LogInfoNode{
		Value:    value,
		Children: make(map[string]*LogInfoNode),
	}
}

func (node *LogInfoNode) Insert(keys []string, value string) error {
	if len(keys) == 0 {
		return nil
	}

	key := keys[0]
	if key == "" {
		return nil
	}
	if len(keys) > 1 {
		if child, ok := node.Children[key]; ok {
			err := child.Insert(keys[1:], value)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%s has no parent node", key)
		}
	} else {
		child := NewLogInfoNode(value)
		node.Children[key] = child
	}

	return nil
}

func (node *LogInfoNode) Get(key string) string {
	if child, ok := node.Children[key]; ok {
		return child.Value
	}

	return ""
}
