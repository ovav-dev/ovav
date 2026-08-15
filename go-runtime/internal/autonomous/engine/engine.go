// Package engine implements OVAV's autonomous research orchestration.
package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ovav/ovav/internal/autonomous"
	"github.com/ovav/ovav/internal/autonomous/parser"
	"github.com/ovav/ovav/internal/autonomous/scheduler"
	"github.com/ovav/ovav/internal/autonomous/scraper"
	"gopkg.in/yaml.v3"
)

// Engine orchestrates the autonomous research system.
type Engine struct {
	scheduler *scheduler.Scheduler
	scraper   *scraper.Client
	dataDir   string
	targets   []scheduler.Target
}

// Config holds engine configuration.
type Config struct {
	DataDir string
	Timeout time.Duration
}

// New creates a new research engine.
func New(cfg Config) (*Engine, error) {
	if cfg.DataDir == "" {
		cfg.DataDir = ".ovav/intelligence"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Load or create targets
	targets := loadTargets(cfg.DataDir)
	if len(targets) == 0 {
		targets = scheduler.DefaultTargets()
	}

	return &Engine{
		scheduler: scheduler.New(targets),
		scraper:   scraper.New(cfg.Timeout),
		dataDir:   cfg.DataDir,
		targets:   targets,
	}, nil
}

// Run executes a research cycle for all due targets.
func (e *Engine) Run() (*autonomous.ResearchResult, error) {
	start := time.Now()
	result := &autonomous.ResearchResult{
		Timestamp:   start,
		URLsScraped: []string{},
		Findings:    []autonomous.Finding{},
		Changes:     []autonomous.Change{},
		Errors:      []string{},
	}

	var dueTargets []scheduler.Target
	for i := range e.targets {
		if e.scheduler.ShouldRun(&e.targets[i]) {
			dueTargets = append(dueTargets, e.targets[i])
		}
	}

	if len(dueTargets) == 0 {
		return result, nil
	}

	for _, target := range dueTargets {
		result.URLsScraped = append(result.URLsScraped, target.URL)

		findings, changes, err := e.researchTarget(&target)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", target.ID, err))
			continue
		}

		result.Findings = append(result.Findings, findings...)
		result.Changes = append(result.Changes, changes...)

		// Update scheduler
		e.scheduler.UpdateTarget(&target)

		// Update in-memory targets
		for i, t := range e.targets {
			if t.ID == target.ID {
				e.targets[i] = target
				break
			}
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	result.TargetID = "full-cycle"

	// Save findings
	if err := e.saveFindings(result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("save findings: %v", err))
	}

	// Save updated targets
	if err := e.saveTargets(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("save targets: %v", err))
	}

	return result, nil
}

// researchTarget performs research on a single target.
func (e *Engine) researchTarget(target *scheduler.Target) ([]autonomous.Finding, []autonomous.Change, error) {
	var findings []autonomous.Finding
	var changes []autonomous.Change

	// Scrape content
	content, err := e.scraper.Scrape(target)
	if err != nil {
		return nil, nil, fmt.Errorf("scrape: %w", err)
	}

	// Load previous content for change detection
	prevContent := e.loadPreviousContent(target.ID)

	// Parse content
	p := parser.New(target.ID)
	newFindings, _ := p.ParseHTML(content)
	findings = append(findings, newFindings...)

	// Detect changes if we have previous content
	if prevContent != "" {
		newChanges := p.ExtractChanges(prevContent, content)
		changes = append(changes, newChanges...)
	}

	// Save current content for next run
	e.saveContent(target.ID, content)

	return findings, changes, nil
}

// Status returns the current research status.
func (e *Engine) Status() *autonomous.ResearchStatus {
	status := &autonomous.ResearchStatus{
		NextScheduled: e.scheduler.NextScheduled(),
		Running:       false,
	}
	// Convert scheduler.Targets to ResearchTargets
	for _, t := range e.targets {
		status.Targets = append(status.Targets, autonomous.ResearchTarget{
			ID:        t.ID,
			Name:      t.Name,
			URL:       t.URL,
			Frequency: t.Frequency,
			LastRun:   t.LastRun,
			NextRun:   t.NextRun,
			Enabled:   t.Enabled,
		})
	}

	// Count findings
	findingsDir := filepath.Join(e.dataDir, "findings")
	if info, err := os.Stat(findingsDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(findingsDir)
		status.TotalFindings = len(entries)
	}

	return status
}

// GetIntelligenceLayer returns the AI intelligence layer for advanced analysis.
func (e *Engine) GetIntelligenceLayer() *IntelligenceLayer {
	return NewIntelligenceLayer(e)
}

// ListFindings returns all saved findings.
func (e *Engine) ListFindings() ([]autonomous.Finding, error) {
	var allFindings []autonomous.Finding

	findingsDir := filepath.Join(e.dataDir, "findings")
	if _, err := os.Stat(findingsDir); os.IsNotExist(err) {
		return allFindings, nil
	}

	entries, err := os.ReadDir(findingsDir)
	if err != nil {
		return nil, fmt.Errorf("read findings dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(findingsDir, entry.Name()))
		if err != nil {
			continue
		}

		var finding autonomous.Finding
		if err := yaml.Unmarshal(data, &finding); err != nil {
			continue
		}
		allFindings = append(allFindings, finding)
	}

	return allFindings, nil
}

// loadTargets loads research targets from disk.
func loadTargets(dataDir string) []scheduler.Target {
	targetsFile := filepath.Join(dataDir, "targets.yaml")
	data, err := os.ReadFile(targetsFile)
	if err != nil {
		return nil
	}

	var targets []scheduler.Target
	if err := yaml.Unmarshal(data, &targets); err != nil {
		return nil
	}

	return targets
}

// saveTargets saves research targets to disk.
func (e *Engine) saveTargets() error {
	if err := os.MkdirAll(e.dataDir, 0755); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}

	data, err := yaml.Marshal(e.targets)
	if err != nil {
		return fmt.Errorf("marshal targets: %w", err)
	}

	targetsFile := filepath.Join(e.dataDir, "targets.yaml")
	return os.WriteFile(targetsFile, data, 0644)
}

// loadPreviousContent loads cached content for a target.
func (e *Engine) loadPreviousContent(targetID string) string {
	contentFile := filepath.Join(e.dataDir, "cache", targetID+".html")
	data, err := os.ReadFile(contentFile)
	if err != nil {
		return ""
	}
	return string(data)
}

// saveContent caches content for a target.
func (e *Engine) saveContent(targetID, content string) error {
	cacheDir := filepath.Join(e.dataDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	contentFile := filepath.Join(cacheDir, targetID+".html")
	return os.WriteFile(contentFile, []byte(content), 0644)
}

// saveFindings saves research findings to disk.
func (e *Engine) saveFindings(result *autonomous.ResearchResult) error {
	findingsDir := filepath.Join(e.dataDir, "findings")
	if err := os.MkdirAll(findingsDir, 0755); err != nil {
		return fmt.Errorf("mkdir findings dir: %w", err)
	}

	for _, finding := range result.Findings {
		data, err := yaml.Marshal(finding)
		if err != nil {
			continue
		}
		filename := filepath.Join(findingsDir, finding.ID+".yaml")
		os.WriteFile(filename, data, 0644)
	}

	return nil
}

// RunTarget runs research on a specific target.
func (e *Engine) RunTarget(targetID string) (*autonomous.ResearchResult, error) {
	var target *scheduler.Target
	for i := range e.targets {
		if e.targets[i].ID == targetID {
			target = &e.targets[i]
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("unknown target: %s", targetID)
	}

	start := time.Now()
	result := &autonomous.ResearchResult{
		TargetID:    targetID,
		Timestamp:   start,
		URLsScraped: []string{target.URL},
	}

	findings, changes, err := e.researchTarget(target)
	if err != nil {
		result.Errors = []string{err.Error()}
		return result, err
	}

	result.Findings = findings
	result.Changes = changes
	result.DurationMs = time.Since(start).Milliseconds()

	// Save
	if err := e.saveFindings(result); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	return result, nil
}
