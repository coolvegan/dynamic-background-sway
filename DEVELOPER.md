# Developer Guide - dynamic_background

A dynamic desktop background for Sway/Wayland that shows real-time system info (CPU, RAM, disk, network, battery, clock, custom scripts) as widgets on the wallpaper.

## First 10 Minutes: Reading Guide

If you're new to this codebase, read files in this order:

### Minute 1-2: What is a Widget?
**File:** `internal/domain/widget.go`
- A Widget is the core concept: a rectangular area on screen that displays data
- It has a Type (cpu, memory, clock...), Position, Size, Style, and a Value (the text to display)
- The `Dirty` flag tells the renderer "this widget changed, redraw it"

### Minute 3-4: How is everything configured?
**File:** `internal/domain/config.go`
- Config holds all widgets, background settings, and API settings
- Loaded from YAML at startup, can be hot-reloaded via API
- `NewConfig()` validates everything before the app starts

### Minute 5-6: Where does data come from?
**File:** `internal/domain/collector.go`
- The `Collector` interface defines how system data is fetched
- Each widget type maps to one collector (CPU → CPUCollector, etc.)
- `Collect(ctx)` returns `CollectorData` which becomes the widget's `Value`

### Minute 7-8: How does it all connect?
**File:** `internal/application/orchestrator.go`
- The Orchestrator is the "conductor" — it starts everything and keeps it running
- It creates a WidgetManager (widgets ↔ collectors) and a Scheduler (timers)
- The render loop runs every 100ms: get widgets → render → commit to screen

### Minute 9-10: How does it get on screen?
**File:** `internal/infrastructure/wayland/layer.go`
- CGO bridge to the Wayland protocol
- Creates layer-shell surfaces (background layer) for each monitor
- Go renders to an RGBA image → C converts to ARGB → compositor displays it

## Quick Start

```bash
pkill swaybg
./dynamic-background -config config/example.yaml
```

## Architecture (3 layers)

```
┌─────────────────────────────────────────┐
│  Domain (internal/domain)               │
│  Widget, Config, Collector interfaces   │
│  Pure business logic, no I/O            │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Application (internal/application)     │
│  WidgetManager, Scheduler, Orchestrator │
│  Wires components together              │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Infrastructure (internal/infrastructure)│
│  Wayland CGO, Collectors, API, Renderer │
│  Talks to the outside world             │
└─────────────────────────────────────────┘
```

For a detailed architecture with diagrams, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Common Patterns

### 1. Interface + Mock Pattern
Every infrastructure component has an interface in the same package with a Mock implementation:

```go
// Interface
type Collector interface {
    Collect(ctx context.Context) (CollectorData, error)
}

// Mock for testing
type MockCollector struct {
    Data CollectorData
    Err  error
}
```

This allows testing application logic without real system calls or Wayland.

### 2. Factory Functions
All types are created via factory functions that validate inputs:

```go
widget, err := domain.NewWidget(type, position, size, style, interval)
config, err := domain.NewConfig(cfg)
```

Never construct domain types directly — always use the factory.

### 3. Dirty Flag Rendering
Widgets track whether they need re-rendering:

```
Collector updates widget → widget.MarkDirty()
Render loop (100ms)       → checks dirty, renders, widget.MarkClean()
```

### 4. Context-Based Cancellation
All long-running operations accept `context.Context`:

```go
func (s *Scheduler) Start(ctx context.Context) error
func (c *Collector) Collect(ctx context.Context) (CollectorData, error)
```

Cancel the context to stop everything cleanly.

### 5. Shared Config Pointer
The `*domain.Config` is shared between all components. When the API replaces
`cfg.Widgets`, the WidgetManager and Scheduler immediately see the new widgets.
A mutex protects concurrent access.

## Data Flow: API → Render → Screen

Here's how a config change flows through the entire system:

```
1. HTTP PUT /api/v1/config
   ↓
2. api.Server.handleUpdateConfig()
   - Decode JSON, validate, create domain.Widgets
   ↓
3. s.cfg.Widgets = newWidgets  (shared pointer, all components see change)
   ↓
4. orchestrator.UpdateBackgroundConfig(cfg)
   - renderer.SetConfig(cfg)
   - orchestrator.renderFrame(ctx)
   ↓
5. renderFrame()
   - widgetManager.GetAllWidgets()
   - renderer.Render(ctx, renderContext)
     - drawBackground(img, cfg)
     - drawWidget(img, widget) for each widget
   - blitToSHM(img)  (RGBA → ARGB8888 conversion)
   ↓
6. surface.Commit()
   - c_layer_render_monitor() per monitor
   - wl_surface_attach + wl_surface_commit
   ↓
7. Screen updates!
```

## Data Flow: Periodic Widget Update

```
Scheduler (1 goroutine per widget)
   ↓ every widget.Interval (e.g. 1s for CPU)
WidgetManager.UpdateWidget(ctx, widget)
   ↓
collector.Collect(ctx)
   ↓ reads /proc/stat, /proc/meminfo, etc.
widget.Value = formatted data
widget.MarkDirty()
   ↓
Next 100ms render tick picks up the dirty widget
```

## Key Files to Read First

1. `internal/domain/widget.go` - What is a Widget? (types, state, lifecycle)
2. `internal/domain/config.go` - Configuration structure
3. `internal/domain/collector.go` - Collector interface (how data flows in)
4. `internal/application/orchestrator.go` - How everything connects
5. `internal/infrastructure/wayland/layer.go` - Wayland Layer Shell (CGO)

## Wayland Layer Shell (How rendering works)

- Uses `zwlr_layer_shell_v1` protocol to create background surfaces
- One surface per monitor (multi-monitor support, up to 8 outputs)
- Each surface has its own SHM buffer (XRGB8888 format)
- Go renders to RGBA image → blitToSHM converts to XRGB → commit
- CGO handles: connect, registry, surface creation, buffer allocation, commit

## CGO Boundary

```
Go side                          C side
─────────                        ────────
LayerSurface.Connect()    →      c_layer_init()
                                 - wl_display_connect
                                 - registry bind (compositor, shm, layer_shell, outputs)
                                 - create surface per output
                                 - wait for configure callbacks
                                 - create SHM buffers
LayerSurface.Buffer()     →      returns Go slice over C mmap memory
LayerSurface.Commit()     →      c_layer_render_monitor() + c_layer_commit_all()
                                 - convert RGBA → XRGB per monitor
                                 - wl_surface_attach + commit
```

## Collectors

| Collector | Interval | Source |
|-----------|----------|--------|
| CPU | 1s | /proc/stat |
| Memory | 5s | /proc/meminfo |
| Disk | 30s | syscall.Statfs |
| Network | 2s | /proc/net/dev (eth0) |
| Battery | 10s | /sys/class/power_supply |
| Clock | 1s | time.Now() |
| Custom | configurable | exec.Command |

## Widget Types

`cpu`, `memory`, `disk`, `network`, `battery`, `clock`, `uptime`, `temperature`, `custom`

## API

All endpoints are under `/api/v1/`:
- `GET /health` - Health check
- `GET /api/v1/config` - Get current config
- `PUT /api/v1/config` - Hot-reload config
- `GET /api/v1/widgets` - List widgets
- `POST /api/v1/widgets` - Add a widget
- `PUT /api/v1/widgets/{id}` - Update a widget (TODO)
- `DELETE /api/v1/widgets/{id}` - Remove a widget (TODO)
- `GET /api/v1/system` - System info (uptime, widget count)
- `GET /api/v1/ws` - WebSocket for live updates

## Build

```bash
# With Wayland support (requires wayland-client dev headers)
CGO_ENABLED=1 go build -o dynamic-background ./cmd/dynamic-background/

# Without Wayland (PNG output mode, for testing)
CGO_ENABLED=0 go build -o dynamic-background ./cmd/dynamic-background/
```

## Tests

```bash
go test ./...
```

## Troubleshooting

### "failed to connect to wayland display"
- Make sure you're running inside a Wayland session (check `$WAYLAND_DISPLAY`)
- Install wayland-client development headers: `sudo apt install libwayland-dev`
- The wlr-layer-shell protocol must be supported (Sway, Hyprland, River support it)

### "missing required Wayland protocols"
- Your compositor doesn't support `zwlr_layer_shell_v1`
- Try Sway, Hyprland, or River. GNOME/KDE may need extensions.

### "no outputs found"
- No monitors detected. Check `wlr-randr` or `swaymsg -t get_outputs`
- The CGO code has a 100-retry timeout for monitor configuration — if monitors are slow to report, it may time out.

### Widgets not updating
- Check the widget's `Interval` in config — longer intervals mean slower updates
- Check collector errors in the widget's Value (collectors set "error: ..." on failure)
- The render loop runs at 100ms — changes appear within 100ms of collection

### Widgets not appearing on screen
- Check widget Position and Size — widgets outside screen bounds are clipped
- Check widget Style.Color — white text on white background is invisible
- The API may have replaced widgets without triggering a re-render (use PUT /config which does trigger it)

### CGO build fails
- Ensure `pkg-config` can find wayland-client: `pkg-config --cflags wayland-client`
- On NixOS: `nix-shell -p wayland pkg-config`
- On Arch: `sudo pacman -S wayland`

### Running without Wayland (PNG mode)
- Build with `CGO_ENABLED=0`
- Renders to `/tmp/dynamic_background.png` by default (configurable with `-output`)
- Uses `swaymsg output * bg <path> fill` to set the PNG as wallpaper (via BgUpdater)
- Useful for testing rendering without a live Wayland session

### Memory usage is high
- The SHM buffer is `width × height × 4` bytes per monitor
- A 4K monitor uses ~33MB per buffer. With 8 monitors max, that's ~264MB in C memory.
- Go's RGBA buffer is sized to the largest monitor.

### How to add a new collector
1. Create a new file in `internal/infrastructure/collector/` (e.g., `gpu.go`)
2. Implement the `domain.Collector` interface:
   ```go
   type GPUCollector struct{}
   func (c *GPUCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
       // fetch GPU data
       return domain.CollectorData{Value: "NVIDIA RTX 4090"}, nil
   }
   ```
3. Add a new `WidgetType` in `internal/domain/widget.go`
4. Register it in `createCollectors()` in `main_cgo.go`

### How to change rendering
- Modify `internal/infrastructure/renderer/draw.go` for drawing primitives
- Modify `internal/infrastructure/renderer/wayland.go` for the Wayland render pipeline
- Add a new Renderer implementation that satisfies the `renderer.Renderer` interface

## Common Tasks

- Add a new collector: implement `domain.Collector` interface, register in `main_cgo.go`
- Change rendering: modify `internal/infrastructure/renderer/draw.go`
- Add API endpoint: modify `internal/interfaces/api/server.go`
- Change widget layout: modify `internal/infrastructure/renderer/draw.go` drawWidget()
