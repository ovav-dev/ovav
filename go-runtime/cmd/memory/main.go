// Package main implements the ovav memory command.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/ovav/ovav/internal/memory"
)

func main() {
	root := os.Getenv("OVAV_ROOT")
	if root == "" {
		root = "."
	}
	memoryDir := filepath.Join(root, ".ovav", "memory")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "search":
		search(os.Args[2:], memoryDir)
	case "index":
		indexCards(memoryDir)
	case "stats":
		stats(memoryDir)
	case "dedup":
		deduplicate(memoryDir)
	case "rebuild":
		rebuild(memoryDir)
	case "learn":
		learn(memoryDir, os.Args[2:])
	case "insights":
		insights(memoryDir)
	default:
		printUsage()
		os.Exit(1)
	}
}

func search(args []string, memoryDir string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	query := fs.String("query", "", "Search query")
	limit := fs.Int("limit", 10, "Max results")
	hybrid := fs.Bool("hybrid", false, "Use hybrid search")
	tags := fs.String("tags", "", "Filter by tags")
	fs.Parse(args)

	if *query == "" {
		fmt.Fprintln(os.Stderr, "Error: --query is required")
		fs.Usage()
		os.Exit(1)
	}

	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	var results []memory.SearchResult
	var tagList []string
	if *tags != "" {
		tagList = parseTags(*tags)
	}

	if *hybrid || *tags != "" {
		results, err = vs.SearchHybrid(*query, tagList, *limit)
	} else {
		results, err = vs.Search(*query, *limit)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	fmt.Printf("🔍 Semantic Search Results (%d found)\n\n", len(results))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tScore\tCategory\tText\n")
	fmt.Fprintf(w, "--\t-----\t--------\t----\n")
	for _, r := range results {
		text := r.Text
		if len(text) > 60 {
			text = text[:60] + "..."
		}
		fmt.Fprintf(w, "%s\t%.3f\t%s\t%s\n", r.CardID, r.Score, r.Category, text)
	}
	w.Flush()
}

func parseTags(tagsFlag string) []string {
	var tags []string
	parts := splitComma(tagsFlag)
	for _, t := range parts {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func splitComma(s string) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func indexCards(memoryDir string) {
	ledger, err := memory.LoadLedger(filepath.Join(memoryDir, "ledger.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		os.Exit(1)
	}

	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for _, card := range ledger.Cards {
		if err := vs.IndexCard(&card); err != nil {
			continue
		}
		count++
	}

	if err := vs.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Indexed %d cards\n", count)
}

func stats(memoryDir string) {
	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	st := vs.Stats()

	fmt.Println("📊 Vector Store Statistics")
	fmt.Println()
	fmt.Printf("  Total embeddings: %d\n", st.TotalEmbeddings)
	fmt.Printf("  Data directory: %s\n", st.DataDir)

	ledger, err := memory.LoadLedger(filepath.Join(memoryDir, "ledger.yaml"))
	if err == nil {
		fmt.Printf("  Total cards: %d\n", len(ledger.Cards))
	}
}

func deduplicate(memoryDir string) {
	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔄 Deduplicating embeddings (threshold: 0.95)...")

	removed, err := vs.Deduplicate(0.95)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deduplicating: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Removed %d duplicate embeddings\n", removed)
}

func rebuild(memoryDir string) {
	ledger, err := memory.LoadLedger(filepath.Join(memoryDir, "ledger.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		os.Exit(1)
	}

	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔨 Rebuilding index for %d cards...\n", len(ledger.Cards))

	if err := vs.RebuildIndex(ledger); err != nil {
		fmt.Fprintf(os.Stderr, "Error rebuilding index: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Index rebuilt successfully")
}

func learn(memoryDir string, args []string) {
	query := ""
	result := ""
	feedback := 0.5
	context := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-query":
			if i+1 < len(args) {
				query = args[i+1]
				i++
			}
		case "-result":
			if i+1 < len(args) {
				result = args[i+1]
				i++
			}
		case "-feedback":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%f", &feedback)
				i++
			}
		case "-context":
			if i+1 < len(args) {
				context = args[i+1]
				i++
			}
		}
	}

	if query == "" || result == "" {
		fmt.Fprintln(os.Stderr, "Usage: ovav memory learn -query '...' -result '...' [-feedback 0.5] [-context '...']")
		os.Exit(1)
	}

	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	engine := memory.NewLiveLearningEngine(vs)
	engine.RecordInteraction(query, result, feedback, context)

	fmt.Println("✅ Interaction recorded for learning")
	fmt.Printf("   Query: %s\n", query)
	fmt.Printf("   Feedback: %.2f\n", feedback)

	// Show prediction if available
	if context != "" {
		prediction := engine.PredictAction(query, context)
		if prediction != nil {
			fmt.Printf("\n🧠 AI Prediction:\n")
			fmt.Printf("   Action: %s\n", prediction.Action)
			fmt.Printf("   Confidence: %.0f%%\n", prediction.Confidence*100)
			fmt.Printf("   Reasoning: %s\n", prediction.Reasoning)
		}
	}
}

func insights(memoryDir string) {
	vs, err := memory.NewVectorStore(filepath.Join(memoryDir, "vectors"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	engine := memory.NewLiveLearningEngine(vs)
	insightsList := engine.GetInsights()

	if len(insightsList) == 0 {
		fmt.Println("No insights yet. Record more interactions with 'ovav memory learn'.")
		return
	}

	fmt.Printf("🧠 Learning Insights (%d found)\n\n", len(insightsList))

	for i, insight := range insightsList {
		if i >= 10 {
			break
		}
		fmt.Printf("%d. [%s] %s\n", i+1, insight.Type, insight.Description)
		fmt.Printf("   → %s\n\n", insight.Recommendation)
	}

	stats := engine.Stats()
	fmt.Println("📊 Learning Statistics:")
	fmt.Printf("   Total interactions: %d\n", stats.TotalInteractions)
	fmt.Printf("   Patterns detected: %d\n", stats.TotalPatterns)
	fmt.Printf("   High confidence patterns: %d\n", stats.HighConfidencePatterns)
	fmt.Printf("   Average success rate: %.0f%%\n", stats.AverageSuccessRate*100)
}

func printUsage() {
	fmt.Print(`OVAV Memory - Semantic Search System v4.0

Usage:
  ovav memory <command>

Commands:
  search --query "..."   Semantic search across memory cards
  search --hybrid        Use hybrid (semantic + keyword) search
  index                  Index all memory cards
  stats                  Show vector store statistics
  dedup                  Remove duplicate embeddings
  rebuild                Rebuild entire index
  learn                  Record interaction for AI learning
  insights               Show learned patterns and insights

Examples:
  ovav memory search --query "python migration"
  ovav memory search --query "validators" --limit 20
  ovav memory search --query "testing" --tags "security,unit"
  ovav memory search --query "governance" --hybrid
  ovav memory learn -query "how to fix X" -result "do Y" -feedback 0.9 -context "debugging"
  ovav memory stats
  ovav memory insights
`)
}
