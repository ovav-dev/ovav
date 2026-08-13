package project

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestOVAVAdaptiveThemeContrast(t *testing.T) {
	path := filepath.Join("..", "..", "..", "clients", "opencode", "themes", "ovav.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Theme map[string]map[string]string `json:"theme"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	for _, mode := range []string{"dark", "light"} {
		background := document.Theme["background"][mode]
		for _, token := range []string{
			"text", "textMuted", "textSecondary", "primary", "secondary", "accent",
			"error", "warning", "success", "info", "diffAdded", "diffRemoved", "diffContext",
		} {
			ratio := contrastRatio(document.Theme[token][mode], background)
			if ratio < 4.5 {
				t.Errorf("%s/%s contrast %.2f is below WCAG AA", mode, token, ratio)
			}
		}
	}
}

func contrastRatio(foreground, background string) float64 {
	first := relativeLuminance(foreground)
	second := relativeLuminance(background)
	if first < second {
		first, second = second, first
	}
	return (first + 0.05) / (second + 0.05)
}

func relativeLuminance(color string) float64 {
	raw, err := hex.DecodeString(color[1:])
	if err != nil || len(raw) != 3 {
		return 0
	}
	channels := make([]float64, 3)
	for i, value := range raw {
		channel := float64(value) / 255
		if channel <= 0.04045 {
			channels[i] = channel / 12.92
		} else {
			channels[i] = math.Pow((channel+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}
