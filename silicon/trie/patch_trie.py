import re

with open('trie.go', 'r', encoding='utf-8') as f:
    code = f.read()

old_delete = '''	// Edge compression: if child has no value and no edges, remove it
	if !next.hasValue && len(next.edges) == 0 {
		curr.edges = slices.Delete(curr.edges, idx, idx+1)
	} else if !next.hasValue && len(next.edges) == 1 {
		// Merge child edge with single grandchild
		grandChild := next.edges[0].node
		next.prefix += grandChild.prefix
		next.edges = grandChild.edges
		next.value = grandChild.value
		next.hasValue = grandChild.hasValue
	}

	return val, true'''

new_delete = '''	// Edge compression: if child has no value and no edges, remove it
	if !next.hasValue && len(next.edges) == 0 {
		curr.edges = slices.Delete(curr.edges, idx, idx+1)
	}

	// Merge current node if it has no value and only 1 child
	if curr != t.root && !curr.hasValue && len(curr.edges) == 1 {
		child := curr.edges[0].node
		curr.prefix += child.prefix
		curr.edges = child.edges
		curr.value = child.value
		curr.hasValue = child.hasValue
	}

	return val, true'''

code = code.replace(old_delete, new_delete)

old_search_0 = '''		old := curr.value
		curr.hasValue = false

		var zero V

		curr.value = zero

		return old, true'''

new_search_0 = '''		old := curr.value
		curr.hasValue = false

		var zero V

		curr.value = zero

		if curr != t.root && len(curr.edges) == 1 {
			child := curr.edges[0].node
			curr.prefix += child.prefix
			curr.edges = child.edges
			curr.value = child.value
			curr.hasValue = child.hasValue
		}

		return old, true'''

code = code.replace(old_search_0, new_search_0)

with open('trie.go', 'w', encoding='utf-8') as f:
    f.write(code)
