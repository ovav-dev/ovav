package main

import "strings"

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

func (n *NavStack) Path() []string {
	return n.stack
}

func (n *NavStack) Breadcrumb() string {
	if len(n.stack) <= 1 {
		return ""
	}
	parts := make([]string, 0, len(n.stack)-1)
	for _, v := range n.stack[:len(n.stack)-1] {
		parts = append(parts, viewLabel(v))
	}
	return strings.Join(parts, " › ")
}

func viewLabel(view string) string {
	labels := map[string]string{
		ViewWelcome:     "Welcome",
		ViewRoot:        "Menu",
		ViewDashboard:   "Dashboard",
		ViewHealth:      "Health",
		ViewVault:       "Vault",
		ViewInstall:     "Install",
		ViewTailor:      "Tailor",
		ViewCLI:         "CLI",
		ViewSync:        "Sync",
		ViewConfig:      "Config",
		ViewUpdates:     "Updates",
		ViewDetail:      "Detail",
		ViewQuit:        "Quit",
		ViewHelp:        "Help",
		ViewTesting:     "Testing",
		ViewDelegation:  "Delegation",
		ViewResearch:    "Research",
		ViewAdversarial: "Adversarial",
		ViewPerformance: "Performance",
	}
	if label, ok := labels[view]; ok {
		return label
	}
	return view
}
