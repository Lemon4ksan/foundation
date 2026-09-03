// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs

import (
	"sort"
	"strings"
)

// TreeNode represents an individual directory or file node in the virtual hierarchy.
type TreeNode struct {
	Name     string
	IsDir    bool
	Children map[string]*TreeNode
}

// NewTreeNode constructs a new TreeNode.
func NewTreeNode(name string, isDir bool) *TreeNode {
	return &TreeNode{
		Name:     name,
		IsDir:    isDir,
		Children: make(map[string]*TreeNode),
	}
}

// BuildTree constructs a hierarchical directory tree from a list of entry paths and directory flags.
func BuildTree(entries []struct {
	Name  string
	IsDir bool
},
) *TreeNode {
	root := NewTreeNode(".", true)

	for _, entry := range entries {
		clean := CleanPath(entry.Name)
		if clean == "" {
			continue
		}

		parts := strings.Split(clean, "/")
		current := root

		for i, part := range parts {
			if part == "" {
				continue
			}

			isLast := i == len(parts)-1
			isDir := !isLast || entry.IsDir

			child, exists := current.Children[part]
			if !exists {
				child = NewTreeNode(part, isDir)
				current.Children[part] = child
			} else if isDir {
				child.IsDir = true
			}
			current = child
		}
	}

	return root
}

// RenderTree converts a TreeNode hierarchy into a formatted ASCII visual tree.
func RenderTree(root *TreeNode) string {
	if root == nil || len(root.Children) == 0 {
		return ".\n└── (empty archive)"
	}

	var sb strings.Builder
	sb.WriteString(".\n")
	renderNode(root, "", &sb)
	return sb.String()
}

func renderNode(node *TreeNode, prefix string, sb *strings.Builder) {
	keys := make([]string, 0, len(node.Children))
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, name := range keys {
		child := node.Children[name]
		isLast := i == len(keys)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		sb.WriteString(prefix)
		sb.WriteString(connector)
		sb.WriteString(name)
		if child.IsDir {
			sb.WriteString("/")
		}
		sb.WriteString("\n")

		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		renderNode(child, childPrefix, sb)
	}
}
