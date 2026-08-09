package product

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ── Install ──────────────────────────────────────────────────────────────────

type InstallResult struct {
	Mode         string   `json:"mode"`
	ProductDir   string   `json:"product_dir"`
	FilesCopied  int      `json:"files_copied"`
	LinksCreated int      `json:"links_created"`
	Errors       []string `json:"errors,omitempty"`
	Preview      string   `json:"preview,omitempty"`
}

func ProductInstall(ovavRoot string, mode string) (*InstallResult, error) {
	productDir, err := ProductDir()
	if err != nil {
		return nil, err
	}

	switch mode {
	case "dry-run":
		return dryRun(ovavRoot, productDir)
	case "install":
		return doInstall(ovavRoot, productDir)
	case "verify":
		return doVerify(productDir)
	case "uninstall":
		return doUninstall(productDir)
	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}
}

// ── Dry Run ──────────────────────────────────────────────────────────────────

func dryRun(ovavRoot, productDir string) (*InstallResult, error) {
	r := &InstallResult{Mode: "dry-run", ProductDir: productDir}
	var b strings.Builder

	b.WriteString("╔══════════════════════════════════════════════════════╗\n")
	b.WriteString("║  OVAV Product Install — Dry Run                     ║\n")
	b.WriteString("╠══════════════════════════════════════════════════════╣\n")
	b.WriteString(fmt.Sprintf("║  Source: %s\n", truncate(ovavRoot, 42)))
	b.WriteString(fmt.Sprintf("║  Target: %s\n", truncate(productDir, 42)))
	b.WriteString("╠══════════════════════════════════════════════════════╣\n")

	if _, err := os.Stat(filepath.Join(productDir, ".ovav-manifest.json")); err == nil {
		b.WriteString("║  ⚠  Previous installation — would UPDATE            ║\n")
	} else {
		b.WriteString("║  ✨ Fresh installation                               ║\n")
	}
	b.WriteString("╠══════════════════════════════════════════════════════╣\n")

	for _, asset := range PortableAssets() {
		if asset.Source == "" {
			b.WriteString(fmt.Sprintf("║  📄 [generate] %s\n", asset.Target))
			r.FilesCopied++
			continue
		}
		sp := filepath.Join(ovavRoot, asset.Source)
		info, err := os.Stat(sp)
		if err != nil {
			b.WriteString(fmt.Sprintf("║  ❌ [missing]  %s\n", asset.Source))
			continue
		}
		if info.IsDir() {
			b.WriteString(fmt.Sprintf("║  📁 [symlink]  %s/ (%d files)\n", asset.Target, countFiles(sp)))
			r.LinksCreated++
		} else {
			b.WriteString(fmt.Sprintf("║  📄 [copy]     %s\n", asset.Target))
			r.FilesCopied++
		}
		b.WriteString(fmt.Sprintf("║     %s\n", asset.Description))
	}

	b.WriteString("╠══════════════════════════════════════════════════════╣\n")
	b.WriteString("║  🔒 NOT installed (internal OVAV Systems):          ║\n")
	for _, r := range RestrictedAssets() {
		b.WriteString(fmt.Sprintf("║     • %s\n", truncate(r, 48)))
	}
	b.WriteString("╚══════════════════════════════════════════════════════╝\n")

	r.Preview = b.String()
	return r, nil
}

// ── Install ──────────────────────────────────────────────────────────────────

func doInstall(ovavRoot, productDir string) (*InstallResult, error) {
	r := &InstallResult{Mode: "install", ProductDir: productDir}

	if err := os.MkdirAll(productDir, 0755); err != nil {
		return nil, err
	}

	manifest := NewManifest(ovavRoot, "product")

	for _, asset := range PortableAssets() {
		if asset.Source == "" {
			continue
		}
		src := filepath.Join(ovavRoot, asset.Source)
		dst := filepath.Join(productDir, asset.Target)

		info, err := os.Stat(src)
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("missing: %s", asset.Source))
			continue
		}

		if asset.Category == "agents" {
			// Full install: copy ALL agents (projectDir="" = install all)
			if err := installSelectiveAgents(src, dst, "", manifest, r); err != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("selective agents: %v", err))
			}
			continue
		}

		if info.IsDir() {
			if err := symlink(src, dst); err != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("symlink %s: %v", asset.Target, err))
				continue
			}
			manifest.AddSymlink(dst, src, true)
			r.LinksCreated++
		} else {
			if err := copyFile(src, dst); err != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("copy %s: %v", asset.Target, err))
				continue
			}
			_ = manifest.AddEntry(src, dst, asset.Target, asset.Category, true)
			r.FilesCopied++
		}
	}

	// Generate AGENTS.md
	agentsMD := filepath.Join(productDir, "AGENTS.md")
	if err := generateGlobalAGENTS(agentsMD); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("generate AGENTS.md: %v", err))
	} else {
		r.FilesCopied++
	}

	// Generate mimocode.json with model routing
	configPath := filepath.Join(productDir, "mimocode.json")
	if err := generateMimocodeConfig(configPath); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("generate mimocode.json: %v", err))
	} else {
		r.FilesCopied++
	}

	// GOV-009: Build and install Product Cockpit binary
	cockpitDst := filepath.Join(productDir, "product-cockpit")
	if err := buildProductCockpit(ovavRoot, cockpitDst); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("build cockpit: %v", err))
	} else {
		r.FilesCopied++
	}

	if err := SaveManifest(manifest); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("save manifest: %v", err))
	}

	// GOV-007: Write VERSION file for update detection
	if err := WriteVersionFile(); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("write version: %v", err))
	}

	return r, nil
}

// installSelectiveAgents generates clean product agents (zero OVAV Systems memory).
// installSelectiveAgents copies agents to product dir.
// If projectDir is empty, installs ONLY area-level agents (AreasOnly constraint).
// If projectDir is set, uses stack detection for selective install.
func installSelectiveAgents(agentsSrcDir, agentsDstDir, projectDir string, manifest *Manifest, r *InstallResult) error {
	var agentNames []string
	var stack ProjectStack

	if projectDir == "" {
		// Full install: copy ONLY area-level agents (AreasOnly constraint).
		// MiMoCode TAB picker does not honor hidden:true, so installing
		// leads/teams would leak their names in the picker count.
		// Canonical: MimocodeConverter.AreasOnly() = true — same rule here.
		entries, err := os.ReadDir(agentsSrcDir)
		if err != nil {
			return fmt.Errorf("read agents dir: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "area-") {
				agentNames = append(agentNames, entry.Name())
			}
		}
		stack = ProjectStack{Primary: "full"}
	} else {
		stack = DetectProjectStack(projectDir)
		agentNames = SelectAgentsForStack(stack, agentsSrcDir)
	}

	if len(agentNames) == 0 {
		return nil // nothing to install
	}

	if err := os.MkdirAll(agentsDstDir, 0755); err != nil {
		return err
	}

	copied := 0
	for _, name := range agentNames {
		dst := filepath.Join(agentsDstDir, name)

		// Generate a CLEAN product agent — no OVAV Systems context
		content := generateProductAgent(name, stack)
		if err := os.WriteFile(dst, []byte(content), 0644); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("write agent %s: %v", name, err))
			continue
		}

		_ = manifest.AddEntry(dst, dst, "agents/"+name, "agents", true)
		copied++
	}

	r.FilesCopied += copied
	return nil
}

// ── Verify ───────────────────────────────────────────────────────────────────

func doVerify(productDir string) (*InstallResult, error) {
	r := &InstallResult{Mode: "verify", ProductDir: productDir}

	manifest, err := LoadManifest()
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf("no installation found at %s", productDir)
	}

	for _, entry := range manifest.Entries {
		if entry.Source == "" {
			if _, err := os.Stat(entry.Target); err != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("missing: %s", entry.RelPath))
			}
			continue
		}
		h, err := fileHash(entry.Source)
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("source gone: %s", entry.RelPath))
			continue
		}
		if entry.Hash != "" && h != entry.Hash {
			r.Errors = append(r.Errors, fmt.Sprintf("changed: %s", entry.RelPath))
		}
	}

	for _, link := range manifest.Symlinks {
		target, err := os.Readlink(link.Link)
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("broken: %s", link.Link))
			continue
		}
		if _, err := os.Stat(target); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("dangling: %s → %s", link.Link, target))
		}
	}

	if len(r.Errors) == 0 {
		r.FilesCopied = len(manifest.Entries)
		r.LinksCreated = len(manifest.Symlinks)
	}
	return r, nil
}

// ── Uninstall ────────────────────────────────────────────────────────────────

func doUninstall(productDir string) (*InstallResult, error) {
	r := &InstallResult{Mode: "uninstall", ProductDir: productDir}

	manifest, err := LoadManifest()
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf("no installation found")
	}

	for _, link := range manifest.Symlinks {
		if err := os.Remove(link.Link); err != nil && !os.IsNotExist(err) {
			r.Errors = append(r.Errors, err.Error())
		} else {
			r.LinksCreated++
		}
	}

	for _, entry := range manifest.Entries {
		if entry.Target == "" {
			continue
		}
		if err := os.Remove(entry.Target); err != nil && !os.IsNotExist(err) {
			r.Errors = append(r.Errors, err.Error())
		} else {
			r.FilesCopied++
		}
	}

	mp, _ := ManifestPath()
	_ = os.Remove(mp)

	entries, _ := os.ReadDir(productDir)
	if len(entries) == 0 {
		_ = os.Remove(productDir)
	}

	return r, nil
}

// ── Bootstrap ────────────────────────────────────────────────────────────────

type BootstrapResult struct {
	CWD            string   `json:"cwd"`
	ProductDir     string   `json:"product_dir"`
	AgentsLinked   bool     `json:"agents_linked"`
	SkillsLinked   bool     `json:"skills_linked"`
	IdentityCopied bool     `json:"identity_copied"`
	Errors         []string `json:"errors,omitempty"`
}

// Bootstrap creates .mimocode/ in cwd with symlinks to installed OVAV assets.
func Bootstrap(cwd string) (*BootstrapResult, error) {
	productDir, err := ProductDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(filepath.Join(productDir, ".ovav-manifest.json")); err != nil {
		return nil, fmt.Errorf("OVAV Product not installed — run 'ovav product install'")
	}

	r := &BootstrapResult{CWD: cwd, ProductDir: productDir}
	mc := filepath.Join(cwd, ".mimocode")

	if err := os.MkdirAll(mc, 0755); err != nil {
		return nil, err
	}

	// Symlink agents
	if err := symlink(filepath.Join(productDir, "agents"), filepath.Join(mc, "agents")); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("agents: %v", err))
	} else {
		r.AgentsLinked = true
	}

	// Symlink skills
	if err := symlink(filepath.Join(productDir, "skills"), filepath.Join(mc, "skills")); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("skills: %v", err))
	} else {
		r.SkillsLinked = true
	}

	// Copy identity (only if not present)
	dst := filepath.Join(cwd, "OVAV_IDENTITY.md")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if err := copyFile(filepath.Join(productDir, "OVAV_IDENTITY.md"), dst); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("identity: %v", err))
		} else {
			r.IdentityCopied = true
		}
	} else {
		r.IdentityCopied = true
	}

	return r, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func symlink(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source missing: %s", src)
	}
	if _, err := os.Lstat(dst); err == nil {
		_ = os.RemoveAll(dst)
	}
	return os.Symlink(src, dst)
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	info, err := s.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

func countFiles(dir string) int {
	c := 0
	_ = filepath.Walk(dir, func(_ string, fi os.FileInfo, _ error) error {
		if !fi.IsDir() {
			c++
		}
		return nil
	})
	return c
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// BootstrapCWD bootstraps the current working directory with OVAV Product agents.
// Convenience wrapper for `ovav product launch` from Product Cockpit.
func BootstrapCWD() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	_, err = Bootstrap(cwd)
	return err
}

// buildProductCockpit compiles the Product Cockpit TUI binary and installs it.
// GOV-009: Product Cockpit is the end-user TUI for update alerts and CWD bootstrap.
func buildProductCockpit(ovavRoot, dst string) error {
	goRuntime := filepath.Join(ovavRoot, "go-runtime")
	cockpitSrc := filepath.Join(goRuntime, "cmd", "product_cockpit")

	// Check if source exists (may not in test environments)
	if _, err := os.Stat(cockpitSrc); os.IsNotExist(err) {
		return nil // skip — not a full OVAV Systems repo
	}

	// Build to temp location
	tmpBin := filepath.Join(os.TempDir(), "ovav-product-cockpit-build")
	cmd := exec.Command("go", "build", "-o", tmpBin, cockpitSrc)
	cmd.Dir = goRuntime
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build: %w\n%s", err, string(out))
	}

	// Copy to product dir
	src, err := os.Open(tmpBin)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	defer src.Close()
	defer os.Remove(tmpBin)

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, src); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return os.Chmod(dst, 0755)
}
