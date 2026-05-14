package plugin

import (
	"context"
	"testing"
)

// TestMetadata_VersionIsPlaceholder guards against accidental version
// hardcoding. The in-source `pluginVersion` MUST stay at the "dev"
// placeholder so the GoReleaser `-X` ldflag (set from the git tag) is the
// only path through which a real version string reaches Metadata().
//
// If you are tempted to change the placeholder here: don't — bump the tag
// instead and let the release workflow inject it. See the
// "Release-Pipeline & Versionierung" section in CLAUDE.md.
func TestMetadata_VersionIsPlaceholder(t *testing.T) {
	p := New()
	m, err := p.Metadata(context.Background())
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if m.Version != "dev" {
		t.Errorf("Metadata.Version = %q, want %q (build-time injection placeholder)", m.Version, "dev")
	}
}

func TestMetadata_NameAndCapabilities(t *testing.T) {
	p := New()
	m, err := p.Metadata(context.Background())
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if m.Name != "personio-dayoff" {
		t.Errorf("Metadata.Name = %q", m.Name)
	}
	if len(m.Capabilities) != 1 {
		t.Errorf("Metadata.Capabilities = %v, want [CapOffHoursProvider]", m.Capabilities)
	}
}
