// Package styles defines the OVAV Cockpit visual theme.
// Modern, premium TUI design with charmbracelet/lipgloss.
package styles

import "github.com/charmbracelet/lipgloss"

// ── OVAV Color Palette — Modern ─────────────────────────────────────

var (
	// Primary: electric indigo (OVAV brand)
	Primary    = lipgloss.Color("#6366F1")
	PrimaryBg  = lipgloss.Color("#4338CA")
	PrimaryDim = lipgloss.Color("#818CF8")

	// Accent: vibrant teal
	Accent   = lipgloss.Color("#14B8A6")
	AccentBg = lipgloss.Color("#0D9488")

	// Status colors — vivid
	Green  = lipgloss.Color("#22C55E")
	Yellow = lipgloss.Color("#EAB308")
	Red    = lipgloss.Color("#EF4444")
	Blue   = lipgloss.Color("#3B82F6")
	Cyan   = lipgloss.Color("#06B6D4")
	Purple = lipgloss.Color("#A855F7")
	Orange = lipgloss.Color("#F97316")

	// Neutrals — modern gray scale
	White   = lipgloss.Color("#F8FAFC")
	Bright  = lipgloss.Color("#E2E8F0")
	Gray    = lipgloss.Color("#94A3B8")
	Dark    = lipgloss.Color("#1E293B")
	Darker  = lipgloss.Color("#0F172A")
	Darkest = lipgloss.Color("#020617")
	Muted   = lipgloss.Color("#64748B")
)

// ── Foreground-only shortcuts ──────────────────────────────────────

var (
	PrimaryFg    = lipgloss.NewStyle().Foreground(Primary)
	PrimaryDimFg = lipgloss.NewStyle().Foreground(PrimaryDim)
	BlueFg       = lipgloss.NewStyle().Foreground(Blue)
	MutedFg      = lipgloss.NewStyle().Foreground(Muted)
	GreenFg      = lipgloss.NewStyle().Foreground(Green)
	RedFg        = lipgloss.NewStyle().Foreground(Red)
	YellowFg     = lipgloss.NewStyle().Foreground(Yellow)
	CyanFg       = lipgloss.NewStyle().Foreground(Cyan)
	PurpleFg     = lipgloss.NewStyle().Foreground(Purple)
	WhiteFg      = lipgloss.NewStyle().Foreground(White)
	BrightFg     = lipgloss.NewStyle().Foreground(Bright)
	OrangeFg     = lipgloss.NewStyle().Foreground(Orange)
	AccentFg     = lipgloss.NewStyle().Foreground(Accent)
)

// ── Base Styles ─────────────────────────────────────────────────────

var (
	// App container
	App = lipgloss.NewStyle().
		Padding(0, 1)

	// Title bar — modern gradient feel
	TitleBar = lipgloss.NewStyle().
			Background(PrimaryBg).
			Foreground(White).
			Padding(0, 2).
			Bold(true)

	// Section header — bold with accent underline
	Header = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	// Selected item — vivid highlight with background
	Selected = lipgloss.NewStyle().
			Foreground(Darkest).
			Background(Primary).
			Padding(0, 1).
			Bold(true)

	// Unselected item — subtle
	Unselected = lipgloss.NewStyle().
			Foreground(Gray).
			Padding(0, 1)

	// Hover — brighter than unselected, no background
	Hover = lipgloss.NewStyle().
		Foreground(Bright).
		Padding(0, 1)

	// ── Badges ─────────────────────────────────────────────────────

	SuccessBadge = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	WarningBadge = lipgloss.NewStyle().
			Foreground(Yellow).
			Bold(true)

	ErrorBadge = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	// ── Progress bars ──────────────────────────────────────────────

	ProgressFill = lipgloss.NewStyle().
			Background(Green)

	ProgressEmpty = lipgloss.NewStyle().
			Background(Dark)

	// ── Help text ──────────────────────────────────────────────────

	Help = lipgloss.NewStyle().
		Foreground(Muted).
		Italic(true)

	// ── Borders — modern rounded with accent colors ────────────────

	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(1, 2)

	InfoBox = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(Blue).
		Padding(1, 2)

	// ── Border containers ──────────────────────────────────────────

	GreenBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Green).
			Padding(1, 2)

	YellowBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Yellow).
			Padding(1, 2)

	BlueBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(1, 2)

	PrimaryBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)

	AccentBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			Padding(1, 2)

	YellowBorderCompact = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Yellow).
				Padding(0, 2)

	GreenBorderCompact = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Green).
				Padding(0, 1)

	YellowBorderCompactPad = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Yellow).
				Padding(0, 1)

	PurpleHelpBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Purple).
				Padding(0, 2)

	PrimaryHelpBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 2)

	// ── Text-only styles ───────────────────────────────────────────

	LogoStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	CyanItalic = lipgloss.NewStyle().
			Foreground(Cyan).
			Italic(true)

	BoldWhite = lipgloss.NewStyle().
			Foreground(White).
			Bold(true)

	MutedItalic = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	PurpleCategory = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true).
			Padding(0, 1)

	ActiveStage = lipgloss.NewStyle().
			Foreground(White).
			Bold(true)

	ActiveButton = lipgloss.NewStyle().
			Foreground(White).
			Background(Primary).
			Padding(0, 3).
			Bold(true)

	InactiveButton = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 3)

	KVKey = lipgloss.NewStyle().
		Foreground(Muted).
		Width(14)

	KVValue = lipgloss.NewStyle().
		Foreground(Bright)

	CardHeader = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1)

	GreenBorderLarge = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Green).
				Padding(2, 4)

	// ── Modern additions ───────────────────────────────────────────

	// Separator line
	Separator = lipgloss.NewStyle().
			Foreground(Dark)

	// Tag/badge for status
	TagGreen = lipgloss.NewStyle().
			Foreground(Green).
			Background(Dark).
			Padding(0, 1).
			Bold(true)

	TagYellow = lipgloss.NewStyle().
			Foreground(Yellow).
			Background(Dark).
			Padding(0, 1).
			Bold(true)

	TagRed = lipgloss.NewStyle().
		Foreground(Red).
		Background(Dark).
		Padding(0, 1).
		Bold(true)

	TagBlue = lipgloss.NewStyle().
		Foreground(Blue).
		Background(Dark).
		Padding(0, 1).
		Bold(true)

	TagPurple = lipgloss.NewStyle().
			Foreground(Purple).
			Background(Dark).
			Padding(0, 1).
			Bold(true)

	// Status indicator with color dot
	StatusActive = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	StatusInactive = lipgloss.NewStyle().
			Foreground(Muted)

	StatusWarning = lipgloss.NewStyle().
			Foreground(Yellow).
			Bold(true)

	// Modern card with shadow feel
	CardModern = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryDim).
			Padding(1, 2).
			MarginBottom(1)

	// Action button
	ActionBtn = lipgloss.NewStyle().
			Foreground(White).
			Background(AccentBg).
			Padding(0, 2).
			Bold(true)

	// Danger button
	DangerBtn = lipgloss.NewStyle().
			Foreground(White).
			Background(Red).
			Padding(0, 2).
			Bold(true)

	// Subtle text
	Subtle = lipgloss.NewStyle().
		Foreground(Muted).
		Faint(true)

	// Tab bar — flat, no background boxes (aligns with uniform header)
	TabActive = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Underline(true).
			Padding(0, 1)

	TabInactive = lipgloss.NewStyle().
		Foreground(Muted).
		Padding(0, 1)

	Breadcrumb = lipgloss.NewStyle().
		Foreground(PrimaryDim).
		Italic(true)
)
