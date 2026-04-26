# Architecture - dynamic_background

## System Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           USER INTERFACE                                 │
│                                                                          │
│  ┌─────────────────┐    ┌─────────────────────────────────────────────┐ │
│  │  config.yaml    │    │  HTTP API / WebSocket (optional)            │ │
│  │  (YAML config)  │    │  GET/PUT /api/v1/config, /widgets, /system  │ │
│  └────────┬────────┘    │  WS /api/v1/ws for live updates             │ │
│           │             └──────────────────────┬──────────────────────┘ │
│           │                                    │                        │
│           ▼                                    ▼                        │
├──────────────────────────────────────────────────────────────────────────┤
│                        APPLICATION LAYER                                 │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                      Orchestrator                                   │ │
│  │  ┌──────────────────┐         ┌──────────────────────────────────┐ │ │
│  │  │  WidgetManager   │◄────────│  Scheduler (per-widget timers)   │ │ │
│  │  │  (widget ↔       │         │  - 1 goroutine per widget        │ │ │
│  │  │   collector map) │         │  - individual intervals          │ │ │
│  │  └────────┬─────────┘         └──────────────┬───────────────────┘ │ │
│  │           │                                   │                     │ │
│  │           ▼                                   ▼                     │ │
│  │  ┌──────────────────┐         ┌──────────────────────────────────┐ │ │
│  │  │  renderLoop      │────────►│  renderFrame()                   │ │ │
│  │  │  (100ms ticker)  │         │  1. Get all widgets              │ │ │
│  │  └──────────────────┘         │  2. Build RenderContext          │ │ │
│  │                               │  3. Renderer.Render()            │ │ │
│  │                               │  4. Surface.Commit()             │ │ │
│  │                               │  5. Mark widgets clean           │ │ │
│  │                               └──────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────────────┤
│                        DOMAIN LAYER (pure Go, no I/O)                    │
│                                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐  │
│  │   Widget     │  │   Config     │  │   Collector (interface)      │  │
│  │   - Type     │  │   - Widgets  │  │   Collect(ctx) → Data        │  │
│  │   - Position │  │   - Backgrnd │  │   MockCollector for tests    │  │
│  │   - Size     │  │   - API      │  │   CollectorFunc adapter      │  │
│  │   - Style    │  └──────────────┘  └──────────────────────────────┘  │
│  │   - Dirty    │                                                      │
│  │   - Value    │                                                      │
│  └──────────────┘                                                      │
├──────────────────────────────────────────────────────────────────────────┤
│                     INFRASTRUCTURE LAYER (I/O, CGO)                      │
│                                                                          │
│  ┌───────────────────┐  ┌───────────────────┐  ┌─────────────────────┐ │
│  │   Collectors      │  │   Renderer        │  │   Wayland (CGO)     │ │
│  │   - CPU (/proc)   │  │   - Wayland:      │  │   - LayerSurface    │ │
│  │   - Memory        │  │     SHM blit      │  │   - c_layer_init()  │ │
│  │   - Disk (statfs) │  │   - Image: PNG    │  │   - c_layer_render  │ │
│  │   - Network       │  │   - draw.go:      │  │   - c_layer_commit  │ │
│  │   - Battery       │  │     primitives    │  │   - MAX_OUTPUTS=8   │ │
│  │   - Clock         │  └───────────────────┘  └─────────────────────┘ │
│  │   - Custom (exec) │                                                  │
│  └───────────────────┘  ┌───────────────────┐                          │
│                         │   Config Loader   │                          │
│                         │   (YAML → domain) │                          │
│                         └───────────────────┘                          │
└──────────────────────────────────────────────────────────────────────────┘
```

## Component Relationships

### Dependency Graph

```
main_cgo.go (entry point)
    │
    ├── config.LoadConfig() ──────────► domain.Config
    │
    ├── createCollectors() ───────────► map[WidgetType]Collector
    │       ├── collector.NewCPUCollector()
    │       ├── collector.NewMemoryCollector()
    │       ├── collector.NewDiskCollector()
    │       ├── collector.NewNetworkCollector()
    │       ├── collector.NewBatteryCollector()
    │       ├── collector.NewClockCollector()
    │       └── collector.NewCustomCollector()
    │
    ├── wayland.NewLayerSurface() ────► Surface (CGO impl)
    │       └── s.Connect() ──────────► C: c_layer_init()
    │
    ├── renderer.NewWaylandRenderer() ─► Renderer (CGO impl)
    │
    └── application.NewOrchestrator()
            ├── NewWidgetManager(cfg, collectors)
            │       └── maps WidgetType → Collector
            │
            └── NewScheduler(widgetManager)
                    └── 1 goroutine per widget with Ticker
```

### Interface Contracts

```
┌─────────────────────────────────────────────────────────────┐
│  domain.Collector              domain.Renderer              │
│  ────────────────              ───────────────              │
│  Collect(ctx) → Data           Render(ctx, ctx) → error     │
│                                Clear() → error              │
│       ▲                                ▲                    │
│       │                                │                    │
│  ┌────┴──────────┐            ┌────────┴──────────┐         │
│  │ CPUCollector  │            │ WaylandRenderer   │         │
│  │ MemoryColl.   │            │ ImageRenderer     │         │
│  │ DiskCollector │            │ MockRenderer      │         │
│  │ NetworkColl.  │            └───────────────────┘         │
│  │ BatteryColl.  │                                          │
│  │ ClockCollector│            ┌───────────────────────┐     │
│  │ CustomColl.   │            │ wayland.Surface       │     │
│  │ MockCollector │            │ ───────────────       │     │
│  │ CollectorFunc │            │ Connect(ctx) → error  │     │
│  └───────────────┘            │ Disconnect() → error  │     │
│                               │ Outputs() → []Output  │     │
│                               │ Buffer() → []byte     │     │
│                               │ Bounds() → Rectangle  │     │
│                               │ Commit() → error      │     │
│                               │ State() → SurfaceState│     │
│                               └──────────┬────────────┘     │
│                                          │                  │
│                               ┌──────────┴────────────┐     │
│                               │ LayerSurface (CGO)    │     │
│                               │ MockSurface (test)    │     │
│                               └───────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Full Request Trace: API Config Change → Screen Update

```
1. HTTP PUT /api/v1/config
   │
   ▼
2. api.Server.handleUpdateConfig()
   ├── Decode JSON request
   ├── Parse intervals (string → time.Duration)
   ├── domain.NewWidget() for each widget (validation)
   └── domain.NewConfig() (validation)
   │
   ▼
3. s.cfg updated (Widgets, Background, API)
   │
   ▼
4. orchestrator.UpdateBackgroundConfig(newCfg.Background, ctx)
   ├── renderer.SetConfig(cfg)  (update background settings)
   └── orchestrator.renderFrame(ctx)
   │
   ▼
5. renderFrame()
   ├── widgetManager.GetAllWidgets()
   ├── NewRenderContext(surface.Bounds())
   ├── rc.SetWidgets(allWidgets)
   │
   ▼
6. renderer.Render(ctx, rc)
   ├── image.NewRGBA(width, height)
   ├── drawBackground(img, cfg)
   │   ├── solid: fill with color
   │   ├── gradient: interpolate top→bottom
   │   └── image: load and draw file
   │
   └── for each widget: drawWidget(img, w)
       ├── draw widget background (if styled)
       ├── draw label ("cpu: ", "memory: ", etc.)
       └── draw value (from widget.Value)
   │
   ▼
7. blitToSHM(img)
   ├── Get SHM buffer from surface (Go slice over C mmap)
   └── Convert RGBA → ARGB8888 pixel-by-pixel
       (R↔B swap, A→0xff)
   │
   ▼
8. surface.Commit()
   ├── For each monitor (C side):
   │   c_layer_render_monitor(idx, rgbaPtr, w, h)
   │   ├── Copy pixels to monitor's SHM buffer
   │   ├── wl_surface_attach(surface, buffer, 0, 0)
   │   ├── wl_surface_damage(surface, 0, 0, w, h)
   │   └── wl_surface_commit(surface)
   │
   └── c_layer_commit_all()
       ├── wl_display_flush(display)
       └── wl_display_dispatch(display)
   │
   ▼
9. All widgets marked clean (w.MarkClean())
   │
   ▼
10. WebSocket broadcast to clients: "config_change"
```

### Periodic Widget Update Flow

```
Scheduler.runWidgetTimer()  (1 goroutine per widget)
    │
    │  Every w.Interval (e.g. 1s for CPU, 30s for disk)
    ▼
WidgetManager.UpdateWidget(ctx, widget)
    │
    ├── Lookup collector: collectors[widget.Type]
    │
    ▼
collector.Collect(ctx)
    │
    ├── CPU:    read /proc/stat, calculate delta
    ├── Memory: read /proc/meminfo
    ├── Disk:   syscall.Statfs("/")
    ├── Network: read /proc/net/dev
    ├── Battery: read /sys/class/power_supply/
    ├── Clock:  time.Now().Format()
    └── Custom: exec.Command(cmd).Output()
    │
    ▼
widget.Value = FormatCollectorData(data)
widget.Data = &data
widget.MarkDirty()
    │
    ▼
(Next 100ms render tick picks up dirty widget)
```

## CGO Boundary Explanation

### Architecture

```
┌──────────────────────────────┐       ┌──────────────────────────────┐
│         GO SIDE              │       │          C SIDE              │
│                              │       │                              │
│  LayerSurface                │       │  static global state:        │
│  ├── Connect() ──────────────┼──────►│    display, compositor,      │
│  │                           │  C    │    shm, layer_shell          │
│  │                           │       │    monitors[MAX_OUTPUTS]     │
│  │                           │       │                              │
│  │                           │       │  c_layer_init()              │
│  │                           │◄──────┤    wl_display_connect(NULL)  │
│  │                           │  int  │    registry bind             │
│  │                           │       │    create surfaces           │
│  │                           │       │    wait for configure        │
│  │                           │       │    create SHM buffers        │
│  │                           │       │                              │
│  ├── Buffer() ───────────────┼───────┤  c_layer_monitor_buffer(i)   │
│  │   returns Go []byte       │  ptr  │    → monitors[i].shm_data    │
│  │   over C mmap memory      │       │    (mmap'd POSIX shm fd)     │
│  │                           │       │                              │
│  ├── Commit() ───────────────┼──────►│  c_layer_render_monitor()    │
│  │                           │  C    │    RGBA → ARGB conversion    │
│  │                           │       │    wl_surface_attach/commit  │
│  │                           │       │                              │
│  │                           │       │  c_layer_commit_all()        │
│  │                           │       │    wl_display_flush/dispatch │
│  │                           │       │                              │
│  └── Disconnect() ───────────┼──────►│  c_layer_cleanup()           │
│                              │  C    │    munmap, close, destroy    │
│                              │       │    wl_display_disconnect     │
└──────────────────────────────┘       └──────────────────────────────┘
```

### Key CGO Details

| Aspect | Detail |
|--------|--------|
| **Build tag** | `//go:build cgo` on CGO files; `//go:build !cgo` for fallback |
| **C includes** | wayland-client.h, wlr-layer-shell protocol headers |
| **Memory** | POSIX shared memory (`shm_open`) + `mmap` for SHM buffers |
| **Pixel format** | Go renders RGBA → C converts to XRGB8888 (WL_SHM_FORMAT_XRGB8888) |
| **Threading** | All C calls from Go are serialized by `sync.Mutex` in LayerSurface |
| **Max outputs** | `MAX_OUTPUTS = 8` (static C array) |
| **Protocol** | `zwlr_layer_shell_v1` (wlr-layer-shell-unstable-v1) |

### Memory Layout (SHM Buffer)

```
C side (mmap'd):                    Go side ([]byte):
┌──────────────────────┐            ┌──────────────────────┐
│  Monitor 0 SHM       │            │  s.rgbaBuf           │
│  width × height × 4  │◄──────────►│  maxWidth × maxH × 4 │
│  format: XRGB8888    │  commit    │  format: RGBA        │
│                      │  copies    │                      │
│  Monitor 1 SHM       │            │                      │
│  width × height × 4  │            │                      │
│  format: XRGB8888    │            │                      │
└──────────────────────┘            └──────────────────────┘

Note: Go renders to a single RGBA buffer sized to the LARGEST monitor.
C copies this buffer to each monitor's SHM, clipping or padding as needed.
```

## Multi-Monitor Rendering Strategy

### Current Approach: Single Buffer, Multiple Outputs

```
┌─────────────────────────────────────────────────────┐
│                    Go Render Target                  │
│                                                     │
│  maxWidth × maxHeight (largest monitor dimensions)  │
│                                                     │
│  ┌─────────────────────┐                            │
│  │                     │                            │
│  │   Background +      │                            │
│  │   Widgets drawn     │                            │
│  │   here (RGBA)       │                            │
│  │                     │                            │
│  └─────────────────────┘                            │
│                                                     │
└──────────────────────┬──────────────────────────────┘
                       │ blitToSHM()
                       │ (RGBA → ARGB conversion)
                       ▼
┌─────────────────────────────────────────────────────┐
│                    C Side (Commit)                   │
│                                                     │
│  For each monitor i:                                │
│  ┌───────────────────────────────────────────┐      │
│  │ c_layer_render_monitor(i, rgbaPtr, w, h)  │      │
│  │                                           │      │
│  │  - Copy pixels to monitor[i].shm_data     │      │
│  │  - Clip if Go buffer smaller than monitor │      │
│  │  - Pad with last pixel if Go buffer larger│      │
│  │  - wl_surface_attach + commit             │      │
│  └───────────────────────────────────────────┘      │
│                                                     │
│  c_layer_commit_all()                               │
│  - wl_display_flush                                 │
│  - wl_display_dispatch                              │
└─────────────────────────────────────────────────────┘
```

### Implications

- **Widgets appear identically on all monitors** (same content, positioned from top-left)
- **Resolution limited to largest monitor** (smaller monitors clip the right/bottom)
- **No per-monitor widget positioning** (widgets use absolute coordinates from 0,0)
- **Single render pass** (efficient, but no per-monitor customization)

### Limitations

1. Widgets at x > smaller_monitor_width are invisible on that monitor
2. No HiDPI/scale-aware rendering (scale factor is read but not used in rendering)
3. Background image may be cropped on monitors with different aspect ratios

## File-by-File Guide

### Entry Points

| File | Purpose |
|------|---------|
| `cmd/dynamic-background/main_cgo.go` | Main entry with CGO (Wayland). Use on real system. |
| `cmd/dynamic-background/main.go` | Fallback entry without CGO (PNG output). Use for testing. |
| `cmd/dynamic-background/bgupdater.go` | Throttled swaybg updater (PNG mode only). |

### Domain Layer (`internal/domain/`)

| File | Purpose | Key Types |
|------|---------|-----------|
| `widget.go` | Widget domain entity | `Widget`, `WidgetType`, `Position`, `Size`, `Style`, `Bounds` |
| `config.go` | Configuration domain entity | `Config`, `BackgroundConfig`, `APIConfig`, `BackgroundType` |
| `collector.go` | Collector interface and helpers | `Collector`, `CollectorData`, `MockCollector`, `CollectorFunc` |

### Application Layer (`internal/application/`)

| File | Purpose | Key Types |
|------|---------|-----------|
| `orchestrator.go` | Wires all components; manages lifecycle | `Orchestrator` |
| `widgetmanager.go` | Maps widgets to collectors; handles updates | `WidgetManager` |
| `scheduler.go` | Per-widget timer goroutines | `Scheduler` |

### Infrastructure Layer (`internal/infrastructure/`)

| File | Purpose | Key Types |
|------|---------|-----------|
| `wayland/layer.go` | CGO Wayland Layer Shell implementation | `LayerSurface` |
| `wayland/surface.go` | Surface interface + MockSurface | `Surface`, `MockSurface`, `Output`, `SurfaceState` |
| `renderer/renderer.go` | Renderer interface + MockRenderer | `Renderer`, `MockRenderer` |
| `renderer/wayland.go` | Wayland SHM renderer (CGO) | `WaylandRenderer` |
| `renderer/image.go` | PNG file renderer (non-CGO) | `ImageRenderer` |
| `renderer/draw.go` | Drawing primitives | `drawBackground`, `drawWidget`, `drawText`, `parseHexColor` |
| `renderer/context.go` | Render state container | `RenderContext` |
| `renderer/font.go` | Font loading utilities | `loadFont`, `parseFontString` |
| `collector/*.go` | System data collectors | `CPUCollector`, `MemoryCollector`, etc. |
| `config/config.go` | YAML config loader | `LoadConfig` |

### Interface Layer (`internal/interfaces/`)

| File | Purpose | Key Types |
|------|---------|-----------|
| `api/server.go` | HTTP/WebSocket API server | `Server` |
| `api/websocket.go` | WebSocket helpers | (methods on Server) |

### Configuration

| File | Purpose |
|------|---------|
| `config/example.yaml` | Example configuration |
| `config.yaml` | Default configuration (used if no -config flag) |
