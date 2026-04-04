// internal/router/jimeng_test.go
package router

import (
	"testing"
)

func TestJimengImageAdapter_Name(t *testing.T) {
	a := NewJimengImageAdapter("fake-ak", "fake-sk", "jimeng_t2i_v40")
	if a.Name() != "jimeng-image" {
		t.Errorf("Name() = %q, want %q", a.Name(), "jimeng-image")
	}
}

func TestJimengImageAdapter_Capabilities(t *testing.T) {
	a := NewJimengImageAdapter("fake-ak", "fake-sk", "jimeng_t2i_v40")
	caps := a.Capabilities()
	if caps.Type != ModelTypeImage {
		t.Errorf("Type = %q, want %q", caps.Type, ModelTypeImage)
	}
	if len(caps.Styles) != 3 {
		t.Errorf("len(Styles) = %d, want 3", len(caps.Styles))
	}
}

func TestJimengVideoAdapter_Name(t *testing.T) {
	a := NewJimengVideoAdapter("fake-ak", "fake-sk", "jimeng_vgfm_i2v_l20")
	if a.Name() != "jimeng-video" {
		t.Errorf("Name() = %q, want %q", a.Name(), "jimeng-video")
	}
}

func TestJimengVideoAdapter_Capabilities(t *testing.T) {
	a := NewJimengVideoAdapter("fake-ak", "fake-sk", "jimeng_vgfm_i2v_l20")
	caps := a.Capabilities()
	if caps.Type != ModelTypeVideo {
		t.Errorf("Type = %q, want %q", caps.Type, ModelTypeVideo)
	}
	if len(caps.Styles) != 3 {
		t.Errorf("len(Styles) = %d, want 3", len(caps.Styles))
	}
}
