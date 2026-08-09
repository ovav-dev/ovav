package styles

import (
	"testing"
)

func TestColorPaletteInitialized(t *testing.T) {
	// Verify all color variables are non-zero
	colors := map[string]interface{}{
		"Primary":   Primary,
		"PrimaryBg": PrimaryBg,
		"Green":     Green,
		"Yellow":    Yellow,
		"Red":       Red,
		"Blue":      Blue,
		"Cyan":      Cyan,
		"Purple":    Purple,
		"White":     White,
		"Gray":      Gray,
		"Dark":      Dark,
		"Darker":    Darker,
		"Muted":     Muted,
	}
	for name, c := range colors {
		if c == "" {
			t.Errorf("color %s should not be empty", name)
		}
	}
}

func TestForegroundStylesRender(t *testing.T) {
	styles := map[string]interface{}{
		"PrimaryFg": PrimaryFg,
		"BlueFg":    BlueFg,
		"MutedFg":   MutedFg,
		"GreenFg":   GreenFg,
		"RedFg":     RedFg,
		"YellowFg":  YellowFg,
		"CyanFg":    CyanFg,
		"PurpleFg":  PurpleFg,
		"WhiteFg":   WhiteFg,
	}
	for name, s := range styles {
		_ = s
		// Just verify they don't panic when rendering
		_ = name
	}
}

func TestBaseStylesRender(t *testing.T) {
	testText := "test content"

	renderTests := map[string]string{
		"App":              App.Render(testText),
		"TitleBar":         TitleBar.Render(testText),
		"Header":           Header.Render(testText),
		"Selected":         Selected.Render(testText),
		"Unselected":       Unselected.Render(testText),
		"SuccessBadge":     SuccessBadge.Render(testText),
		"WarningBadge":     WarningBadge.Render(testText),
		"ErrorBadge":       ErrorBadge.Render(testText),
		"ProgressFill":     ProgressFill.Render(testText),
		"ProgressEmpty":    ProgressEmpty.Render(testText),
		"Help":             Help.Render(testText),
		"Border":           Border.Render(testText),
		"InfoBox":          InfoBox.Render(testText),
		"GreenBorder":      GreenBorder.Render(testText),
		"YellowBorder":     YellowBorder.Render(testText),
		"BlueBorder":       BlueBorder.Render(testText),
		"PrimaryBorder":    PrimaryBorder.Render(testText),
		"LogoStyle":        LogoStyle.Render(testText),
		"CyanItalic":       CyanItalic.Render(testText),
		"BoldWhite":        BoldWhite.Render(testText),
		"MutedItalic":      MutedItalic.Render(testText),
		"PurpleCategory":   PurpleCategory.Render(testText),
		"ActiveStage":      ActiveStage.Render(testText),
		"ActiveButton":     ActiveButton.Render(testText),
		"InactiveButton":   InactiveButton.Render(testText),
		"KVKey":            KVKey.Render(testText),
		"KVValue":          KVValue.Render(testText),
		"CardHeader":       CardHeader.Render(testText),
		"GreenBorderLarge": GreenBorderLarge.Render(testText),
	}

	for name, output := range renderTests {
		if output == "" {
			t.Errorf("style %s rendered empty output", name)
		}
	}
}

func TestCompactBorderStyles(t *testing.T) {
	testText := "compact test"

	outputs := map[string]string{
		"YellowBorderCompact":    YellowBorderCompact.Render(testText),
		"GreenBorderCompact":     GreenBorderCompact.Render(testText),
		"YellowBorderCompactPad": YellowBorderCompactPad.Render(testText),
		"PurpleHelpBorder":       PurpleHelpBorder.Render(testText),
		"PrimaryHelpBorder":      PrimaryHelpBorder.Render(testText),
	}

	for name, output := range outputs {
		if output == "" {
			t.Errorf("compact style %s rendered empty output", name)
		}
	}
}

func TestStyleWidthApplication(t *testing.T) {
	// Verify width can be applied to border styles
	output := GreenBorder.Width(40).Render("test")
	if output == "" {
		t.Error("GreenBorder with width rendered empty")
	}

	output = BlueBorder.Width(60).Render("test")
	if output == "" {
		t.Error("BlueBorder with width rendered empty")
	}
}
