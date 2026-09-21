package hostprojection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSourceNoFollow(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	if err := os.WriteFile(source, []byte("valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	digest, err := validateSourceNoFollow(source, func(content []byte) error {
		called = true
		if string(content) != "valid" {
			t.Fatalf("validator content = %q", content)
		}
		return nil
	})
	if err != nil || !called || digest == "" {
		t.Fatalf("validateSourceNoFollow() digest=%q called=%v err=%v", digest, called, err)
	}

	link := filepath.Join(base, "source-link")
	if err := os.Symlink(source, link); err == nil {
		if _, err := validateSourceNoFollow(link, nil); err == nil {
			t.Fatal("validateSourceNoFollow accepted source symlink")
		}
	}

	realParent := filepath.Join(base, "real-parent")
	if err := os.Mkdir(realParent, 0o755); err != nil {
		t.Fatal(err)
	}
	parentSource := filepath.Join(realParent, "source")
	if err := os.WriteFile(parentSource, []byte("valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	linkedParent := filepath.Join(base, "linked-parent")
	if err := os.Symlink(realParent, linkedParent); err == nil {
		if _, err := validateSourceNoFollow(filepath.Join(linkedParent, "source"), nil); err == nil {
			t.Fatal("validateSourceNoFollow accepted symlink parent")
		}
	}
}
