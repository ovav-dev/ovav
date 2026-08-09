package main

// Navigation stack for view history. Enables proper back navigation
// instead of brittle string assignment.

type NavStack struct {
	stack []string
}

func NewNavStack(root string) NavStack {
	return NavStack{stack: []string{root}}
}

func (n *NavStack) Current() string {
	if len(n.stack) == 0 {
		return ViewWelcome
	}
	return n.stack[len(n.stack)-1]
}

func (n *NavStack) Push(view string) {
	n.stack = append(n.stack, view)
}

func (n *NavStack) Pop() string {
	if len(n.stack) <= 1 {
		return n.Current()
	}
	n.stack = n.stack[:len(n.stack)-1]
	return n.Current()
}

func (n *NavStack) Replace(view string) {
	if len(n.stack) > 0 {
		n.stack[len(n.stack)-1] = view
	} else {
		n.stack = append(n.stack, view)
	}
}

func (n *NavStack) Depth() int {
	return len(n.stack)
}

func (n *NavStack) CanGoBack() bool {
	return len(n.stack) > 1
}
