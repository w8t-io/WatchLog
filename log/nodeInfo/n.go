package nodeInfo

import "fmt"

// LogInfoNode 表示树结构中的一个节点
type LogInfoNode struct {
	Value    string
	Children map[string]*LogInfoNode
}

// NewLogInfoNode 创建一个新的节点
func NewLogInfoNode(value string) *LogInfoNode {
	return &LogInfoNode{
		Value:    value,
		Children: make(map[string]*LogInfoNode),
	}
}

// Insert 插入新值，如果必要会自动创建缺失的父节点
func (node *LogInfoNode) Insert(key string, value string) error {
	if len(key) == 0 {
		return fmt.Errorf("键不能为空")
	}

	node.Children[key] = NewLogInfoNode(value)
	return nil
}

// Get 根据键获取对应的值
func (node *LogInfoNode) Get(key string) string {
	if child, ok := node.Children[key]; ok {
		return child.Value
	}
	return ""
}
