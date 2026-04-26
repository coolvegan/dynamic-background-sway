package wayland

import (
	"context"
	"testing"
)

func TestMockSurface_Connect(t *testing.T) {
	s := NewMockSurface()

	if s.State() != SurfaceStateInitialized {
		t.Errorf("expected state Initialized, got %v", s.State())
	}

	ctx := context.Background()
	if err := s.Connect(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.ConnectCalled {
		t.Error("expected ConnectCalled to be true")
	}

	if s.State() != SurfaceStateRunning {
		t.Errorf("expected state Running, got %v", s.State())
	}
}

func TestMockSurface_Disconnect(t *testing.T) {
	s := NewMockSurface()

	ctx := context.Background()
	_ = s.Connect(ctx)

	if err := s.Disconnect(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.DisconnectCalled {
		t.Error("expected DisconnectCalled to be true")
	}

	if s.State() != SurfaceStateStopped {
		t.Errorf("expected state Stopped, got %v", s.State())
	}
}

func TestMockSurface_Outputs(t *testing.T) {
	s := NewMockSurface()

	outputs := s.Outputs()
	if len(outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(outputs))
	}

	if outputs[0].Name != "mock-output-1" {
		t.Errorf("expected output name 'mock-output-1', got %q", outputs[0].Name)
	}

	if outputs[0].Width != 1920 || outputs[0].Height != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", outputs[0].Width, outputs[0].Height)
	}
}

func TestMockSurface_CreateSurface(t *testing.T) {
	s := NewMockSurface()

	output := Output{Name: "HDMI-1", Width: 2560, Height: 1440, Scale: 1}
	if err := s.CreateSurface(output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.SurfacesCreated != 1 {
		t.Errorf("expected 1 surface created, got %d", s.SurfacesCreated)
	}

	bounds := s.Bounds()
	if bounds.Dx() != 2560 || bounds.Dy() != 1440 {
		t.Errorf("expected bounds 2560x1440, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Buffer should match new dimensions
	buf := s.Buffer()
	expectedLen := 2560 * 1440 * 4
	if len(buf) != expectedLen {
		t.Errorf("expected buffer size %d, got %d", expectedLen, len(buf))
	}
}

func TestMockSurface_Commit(t *testing.T) {
	s := NewMockSurface()

	if err := s.Commit(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.CommitCalled != 1 {
		t.Errorf("expected CommitCalled to be 1, got %d", s.CommitCalled)
	}
}

func TestMockSurface_FrameCallback(t *testing.T) {
	s := NewMockSurface()

	called := make(chan bool, 1)
	s.SetFrameCallback(func() {
		called <- true
	})

	s.TriggerFrameCallback()

	select {
	case <-called:
		// Success
	case <-make(chan struct{}):
		t.Fatal("frame callback was not invoked")
	}
}

func TestMockSurface_Buffer(t *testing.T) {
	s := NewMockSurface()

	buf := s.Buffer()
	if len(buf) == 0 {
		t.Error("expected non-empty buffer")
	}

	// Buffer should be writable
	buf[0] = 0xFF
	if s.Buffer()[0] != 0xFF {
		t.Error("expected buffer to be writable")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Layer != "background" {
		t.Errorf("expected layer 'background', got %q", cfg.Layer)
	}

	if cfg.KeyboardInteract {
		t.Error("expected KeyboardInteract to be false")
	}

	if cfg.ExclusiveZone != 0 {
		t.Errorf("expected ExclusiveZone 0, got %d", cfg.ExclusiveZone)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid background layer",
			cfg:     Config{Layer: "background"},
			wantErr: false,
		},
		{
			name:    "valid bottom layer",
			cfg:     Config{Layer: "bottom"},
			wantErr: false,
		},
		{
			name:    "valid top layer",
			cfg:     Config{Layer: "top"},
			wantErr: false,
		},
		{
			name:    "valid overlay layer",
			cfg:     Config{Layer: "overlay"},
			wantErr: false,
		},
		{
			name:    "invalid layer",
			cfg:     Config{Layer: "invalid"},
			wantErr: true,
		},
		{
			name:    "empty layer",
			cfg:     Config{Layer: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSurfaceState_Values(t *testing.T) {
	states := []SurfaceState{
		SurfaceStateInitialized,
		SurfaceStateRunning,
		SurfaceStateStopped,
	}

	for i, state := range states {
		if int(state) != i {
			t.Errorf("expected state value %d, got %d", i, state)
		}
	}
}
