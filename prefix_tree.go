package main

type TrieNode struct {
	children map[string]*TrieNode
	isEnd    bool
}

type PrefixTree struct {
	root *TrieNode
}

func NewPrefixTree() *PrefixTree {
	return &PrefixTree{
		root: &TrieNode{
			children: make(map[string]*TrieNode),
			isEnd:    false,
		},
	}
}

func (t *PrefixTree) Insert(stringSlice []string) {
	node := t.root
	for _, s := range stringSlice {
		if _, exists := node.children[s]; !exists {
			node.children[s] = &TrieNode{
				children: make(map[string]*TrieNode),
				isEnd:    false,
			}
		}
		node = node.children[s]
	}
	node.isEnd = true
}

func (t *PrefixTree) HasPrefix(prefixSlice []string) bool {
	node := t.root
	for _, s := range prefixSlice {
		if _, exists := node.children[s]; !exists {
			return false
		}
		node = node.children[s]
	}
	return true
}

/* Sample usage:

tree := NewPrefixTree()
tree.Insert([]string{"a", "b", "c"})
tree.Insert([]string{"a", "b", "d"})
tree.Insert([]string{"a", "b", "e"})

fmt.Println(tree.FindPrefixes([]string{"a", "b"}))
*/
