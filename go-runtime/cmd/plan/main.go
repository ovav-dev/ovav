package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

type Task struct {
	ID          string    `yaml:"id"`
	Title       string    `yaml:"title"`
	Description string    `yaml:"description,omitempty"`
	Status      string    `yaml:"status"`
	Priority    string    `yaml:"priority"`
	Layer       int       `yaml:"layer,omitempty"`
	Assignee    string    `yaml:"assignee,omitempty"`
	Tags        []string  `yaml:"tags,omitempty"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
	DueDate     string    `yaml:"due_date,omitempty"`
}

type Plan struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description,omitempty"`
	Tasks       []Task    `yaml:"tasks"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
}

func main() {
	root := os.Getenv("OVAV_ROOT")
	if root == "" {
		root = "."
	}
	planDir := filepath.Join(root, ".ovav", "plans")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initPlan(planDir)
	case "add":
		addTask(os.Args[2:], planDir)
	case "list":
		listTasks(os.Args[2:], planDir)
	case "done":
		markDone(os.Args[2:], planDir)
	case "progress":
		showProgress(planDir)
	default:
		printUsage()
		os.Exit(1)
	}
}

func initPlan(planDir string) {
	if err := os.MkdirAll(planDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating plan dir: %v\n", err)
		os.Exit(1)
	}

	plan := &Plan{
		Name:        "OVAV Stabilization Plan",
		Description: "Layered implementation plan for OVAV stabilization",
		Tasks:       []Task{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	data, err := yaml.Marshal(plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling plan: %v\n", err)
		os.Exit(1)
	}

	planFile := filepath.Join(planDir, "main.yaml")
	if err := os.WriteFile(planFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing plan: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Plan initialized at .ovav/plans/main.yaml")
}

func addTask(args []string, planDir string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ovav plan add -title 'Task title' [-desc 'desc'] [-priority low|medium|high|critical] [-layer N]")
		os.Exit(1)
	}

	title := ""
	desc := ""
	priority := "medium"
	layer := 0

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-title", "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "-desc", "--desc", "-description":
			if i+1 < len(args) {
				desc = args[i+1]
				i++
			}
		case "-priority", "--priority":
			if i+1 < len(args) {
				priority = args[i+1]
				i++
			}
		case "-layer", "--layer":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &layer)
				i++
			}
		}
	}

	if title == "" {
		fmt.Fprintln(os.Stderr, "Title is required")
		os.Exit(1)
	}

	plan := loadPlan(planDir)

	task := Task{
		ID:          fmt.Sprintf("task-%d", len(plan.Tasks)+1),
		Title:       title,
		Description: desc,
		Status:      "todo",
		Priority:    priority,
		Layer:       layer,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        []string{},
	}

	plan.Tasks = append(plan.Tasks, task)
	savePlan(plan, planDir)

	fmt.Printf("✅ Task added: %s\n", task.ID)
}

func listTasks(args []string, planDir string) {
	layer := 0
	status := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--layer", "-layer":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &layer)
				i++
			}
		case "--status", "-status":
			if i+1 < len(args) {
				status = args[i+1]
				i++
			}
		}
	}

	plan := loadPlan(planDir)

	fmt.Println("📋 OVAV Plan Tasks")
	fmt.Println()

	if len(plan.Tasks) == 0 {
		fmt.Println("  No tasks found. Run 'ovav plan init' first.")
		return
	}

	tasks := plan.Tasks
	if layer > 0 {
		var filtered []Task
		for _, t := range tasks {
			if t.Layer == layer {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	if status != "" {
		var filtered []Task
		for _, t := range tasks {
			if t.Status == status {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	sort.Slice(tasks, func(i, j int) bool {
		statusOrder := map[string]int{"todo": 0, "in-progress": 1, "done": 2}
		if statusOrder[tasks[i].Status] != statusOrder[tasks[j].Status] {
			return statusOrder[tasks[i].Status] < statusOrder[tasks[j].Status]
		}
		priorityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return priorityOrder[tasks[i].Priority] < priorityOrder[tasks[j].Priority]
	})

	for _, t := range tasks {
		statusIcon := statusIcon(t.Status)
		priorityIcon := priorityIcon(t.Priority)
		layerTag := ""
		if t.Layer > 0 {
			layerTag = fmt.Sprintf(" [L%d]", t.Layer)
		}
		fmt.Printf("  %s %s %s%s\n", statusIcon, priorityIcon, t.Title, layerTag)
		fmt.Printf("     ID: %s | Priority: %s | Status: %s\n", t.ID, t.Priority, t.Status)
	}

	fmt.Printf("\n  Total: %d tasks\n", len(tasks))
}

func markDone(args []string, planDir string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ovav plan done <task-id>")
		os.Exit(1)
	}

	taskID := args[0]
	plan := loadPlan(planDir)

	found := false
	for i := range plan.Tasks {
		if plan.Tasks[i].ID == taskID {
			plan.Tasks[i].Status = "done"
			plan.Tasks[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "Task not found: %s\n", taskID)
		os.Exit(1)
	}

	savePlan(plan, planDir)
	fmt.Printf("✅ Task %s marked as done\n", taskID)
}

func showProgress(planDir string) {
	plan := loadPlan(planDir)

	var total, done, inProgress, todo int
	for _, t := range plan.Tasks {
		total++
		switch t.Status {
		case "done":
			done++
		case "in-progress":
			inProgress++
		default:
			todo++
		}
	}

	fmt.Println("📊 OVAV Plan Progress")
	fmt.Println()
	fmt.Printf("  Total Tasks:  %d\n", total)
	if total > 0 {
		fmt.Printf("  ✅ Done:      %d (%.0f%%)\n", done, float64(done)/float64(total)*100)
		fmt.Printf("  🔄 In Progress: %d (%.0f%%)\n", inProgress, float64(inProgress)/float64(total)*100)
		fmt.Printf("  ⬜ Todo:      %d (%.0f%%)\n", todo, float64(todo)/float64(total)*100)
	}

	if total > 0 {
		fmt.Println()
		fmt.Println("  Progress Bar:")
		barLen := 30
		doneLen := int(float64(done) / float64(total) * float64(barLen))
		fmt.Print("  [")
		for i := 0; i < barLen; i++ {
			if i < doneLen {
				fmt.Print("█")
			} else {
				fmt.Print("░")
			}
		}
		fmt.Println("]")
	}

	type layerStats struct {
		total int
		done  int
	}
	layerProgress := make(map[int]layerStats)
	for _, t := range plan.Tasks {
		stats := layerProgress[t.Layer]
		stats.total++
		if t.Status == "done" {
			stats.done++
		}
		layerProgress[t.Layer] = stats
	}

	if len(layerProgress) > 1 {
		fmt.Println()
		fmt.Println("  Progress by Layer:")
		var layers []int
		for l := range layerProgress {
			layers = append(layers, l)
		}
		sort.Ints(layers)

		for _, l := range layers {
			if l == 0 {
				continue
			}
			p := layerProgress[l]
			if p.total > 0 {
				fmt.Printf("    Layer %d: %d/%d done\n", l, p.done, p.total)
			}
		}
	}
}

func loadPlan(planDir string) *Plan {
	planFile := filepath.Join(planDir, "main.yaml")
	data, err := os.ReadFile(planFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Plan{
				Name:      "OVAV Plan",
				Tasks:     []Task{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
		}
		fmt.Fprintf(os.Stderr, "Error reading plan: %v\n", err)
		os.Exit(1)
	}

	var plan Plan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing plan: %v\n", err)
		os.Exit(1)
	}

	return &plan
}

func savePlan(plan *Plan, planDir string) {
	plan.UpdatedAt = time.Now()
	data, err := yaml.Marshal(plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling plan: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(planDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating plan dir: %v\n", err)
		os.Exit(1)
	}

	planFile := filepath.Join(planDir, "main.yaml")
	if err := os.WriteFile(planFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing plan: %v\n", err)
		os.Exit(1)
	}
}

func statusIcon(status string) string {
	switch status {
	case "done":
		return "✅"
	case "in-progress":
		return "🔄"
	default:
		return "⬜"
	}
}

func priorityIcon(priority string) string {
	switch priority {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	default:
		return "🟢"
	}
}

func printUsage() {
	fmt.Print(`OVAV Plan - Project Management CLI

Usage:
  ovav plan <command>

Commands:
  init              Initialize a new plan
  add               Add a new task
  list              List all tasks
  list --layer N    Filter by layer
  list --status done Filter by status
  done <task-id>    Mark task as done
  progress          Show progress

Examples:
  ovav plan init
  ovav plan add -title "Fix bug" -priority high -layer 1
  ovav plan list --layer 3
  ovav plan done task-1
  ovav plan progress
`)
}

var _ = json.Marshal
