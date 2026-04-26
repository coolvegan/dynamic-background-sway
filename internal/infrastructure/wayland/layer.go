//go:build cgo

// Package wayland provides implementations of the Surface interface for
// managing Wayland layer shell surfaces.
//
// This file (layer.go) is the CGO bridge between Go and the Wayland protocol.
// It contains both the C code (inline in the import "C" block) and the Go
// wrapper (LayerSurface) that exposes Wayland functionality to the rest of
// the application.
//
// Why it exists:
//   Wayland requires clients to use the wlroots layer-shell protocol to create
//   background surfaces that sit behind all windows. This is the only way to
//   draw a desktop wallpaper in Wayland (unlike X11 where you can draw to the
//   root window). The C code handles:
//   - Connecting to the Wayland display (wl_display_connect)
//   - Binding to required protocols (compositor, shm, layer_shell, outputs)
//   - Creating layer surfaces for each connected monitor
//   - Allocating POSIX shared memory buffers for pixel data
//   - Committing surfaces to the compositor for display
//
// How it connects:
//   - LayerSurface implements the Surface interface (defined in surface.go)
//   - main_cgo.go creates LayerSurface and calls Connect() at startup
//   - Connect() calls c_layer_init() which does all the Wayland setup
//   - Buffer() returns a Go []byte slice over the C mmap'd SHM memory
//   - Commit() calls c_layer_render_monitor() per monitor (RGBA→ARGB convert)
//     then c_layer_commit_all() to flush to the compositor
//   - Orchestrator calls Commit() after each render frame
//   - Disconnect() calls c_layer_cleanup() on shutdown
//
// C functions exposed to Go:
//   c_layer_init()          - Connect, bind protocols, create surfaces/buffers
//   c_layer_monitor_*()     - Query monitor properties (width, height, buffer)
//   c_layer_render_monitor() - Copy RGBA pixels to monitor's SHM buffer
//   c_layer_commit_all()    - Flush display and dispatch events
//   c_layer_cleanup()       - Destroy all resources and disconnect
//
// Key concept: The C code maintains static global state (display, monitors
// array). This means only one LayerSurface can exist at a time. The Go side
// protects access with sync.RWMutex.
package wayland

/*
#cgo pkg-config: wayland-client
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <poll.h>
#include <sys/mman.h>
#include <wayland-client.h>
#include "c/wlr-layer-shell-unstable-v1-client-protocol.h"
#include "c/wlr-layer-shell-unstable-v1-protocol.c"
#include "c/xdg-shell-protocol.c"

#define MAX_OUTPUTS 8

static struct wl_display *display = NULL;
static struct wl_compositor *compositor = NULL;
static struct wl_shm *shm = NULL;
static struct zwlr_layer_shell_v1 *layer_shell = NULL;

typedef struct {
    struct wl_output *output;
    struct wl_surface *surface;
    struct zwlr_layer_surface_v1 *layer_surface;
    struct wl_buffer *buffer;
    unsigned char *shm_data;
    int shm_fd;
    size_t shm_size;
    int width, height;
    int configured;
    char name[32];
    int x, y;
    int scale;
} MonitorSurface;

static MonitorSurface monitors[MAX_OUTPUTS];
static int monitor_count = 0;
static int running = 1;

static void layer_configure(void *data, struct zwlr_layer_surface_v1 *s,
    uint32_t serial, uint32_t w, uint32_t h) {
    for (int i = 0; i < monitor_count; i++) {
        if (monitors[i].layer_surface == s) {
            monitors[i].width = w;
            monitors[i].height = h;
            monitors[i].configured = 1;
            zwlr_layer_surface_v1_ack_configure(s, serial);
            return;
        }
    }
}

static void layer_closed(void *data, struct zwlr_layer_surface_v1 *s) {
    running = 0;
}

static const struct zwlr_layer_surface_v1_listener layer_listener = {
    .configure = layer_configure,
    .closed = layer_closed,
};

static void output_geometry(void *data, struct wl_output *output,
    int32_t x, int32_t y, int32_t physical_width, int32_t physical_height,
    int32_t subpixel, const char *make, const char *model, int32_t transform) {
    for (int i = 0; i < monitor_count; i++) {
        if (monitors[i].output == output) {
            monitors[i].x = x;
            monitors[i].y = y;
            snprintf(monitors[i].name, sizeof(monitors[i].name), "%s %s", make, model);
            return;
        }
    }
}

static void output_mode(void *data, struct wl_output *output,
    uint32_t flags, int32_t width, int32_t height, int32_t refresh) {
    if (flags & WL_OUTPUT_MODE_CURRENT) {
        for (int i = 0; i < monitor_count; i++) {
            if (monitors[i].output == output) {
                monitors[i].width = width;
                monitors[i].height = height;
                return;
            }
        }
    }
}

static void output_done(void *data, struct wl_output *output) {}
static void output_scale(void *data, struct wl_output *output, int32_t scale) {
    for (int i = 0; i < monitor_count; i++) {
        if (monitors[i].output == output) {
            monitors[i].scale = scale;
            return;
        }
    }
}

static const struct wl_output_listener output_listener = {
    .geometry = output_geometry,
    .mode = output_mode,
    .done = output_done,
    .scale = output_scale,
};

static void registry_global(void *data, struct wl_registry *reg,
    uint32_t id, const char *interface, uint32_t version) {
    if (strcmp(interface, wl_compositor_interface.name) == 0)
        compositor = wl_registry_bind(reg, id, &wl_compositor_interface, 4);
    else if (strcmp(interface, wl_shm_interface.name) == 0)
        shm = wl_registry_bind(reg, id, &wl_shm_interface, 1);
    else if (strcmp(interface, zwlr_layer_shell_v1_interface.name) == 0)
        layer_shell = wl_registry_bind(reg, id, &zwlr_layer_shell_v1_interface, 1);
    else if (strcmp(interface, wl_output_interface.name) == 0 && monitor_count < MAX_OUTPUTS) {
        uint32_t ver = version;
        if (ver > 3) ver = 3;
        monitors[monitor_count].output = wl_registry_bind(reg, id, &wl_output_interface, ver);
        monitors[monitor_count].scale = 1;
        wl_output_add_listener(monitors[monitor_count].output, &output_listener, NULL);
        monitor_count++;
    }
}

static void registry_remove(void *data, struct wl_registry *reg, uint32_t id) {}

static const struct wl_registry_listener reg_listener = {
    .global = registry_global,
    .global_remove = registry_remove,
};

int c_layer_init(void) {
    memset(monitors, 0, sizeof(monitors));
    monitor_count = 0;
    
    display = wl_display_connect(NULL);
    if (!display) return -1;
    
    struct wl_registry *reg = wl_display_get_registry(display);
    wl_registry_add_listener(reg, &reg_listener, NULL);
    wl_display_roundtrip(display);
    wl_display_roundtrip(display);
    
    if (!compositor || !shm || !layer_shell) return -2;
    if (monitor_count == 0) return -3;
    
    // Create surface + layer for each monitor
    for (int i = 0; i < monitor_count; i++) {
        MonitorSurface *m = &monitors[i];
        m->surface = wl_compositor_create_surface(compositor);
        m->layer_surface = zwlr_layer_shell_v1_get_layer_surface(
            layer_shell, m->surface, m->output, ZWLR_LAYER_SHELL_V1_LAYER_BACKGROUND, "dynamic_background");
        
        zwlr_layer_surface_v1_add_listener(m->layer_surface, &layer_listener, NULL);
        zwlr_layer_surface_v1_set_anchor(m->layer_surface, 15);
        zwlr_layer_surface_v1_set_keyboard_interactivity(m->layer_surface, 0);
        zwlr_layer_surface_v1_set_exclusive_zone(m->layer_surface, 0);
        wl_surface_commit(m->surface);
    }
    
    // Wait for all monitors to be configured
    for (int retries = 0; retries < 100; retries++) {
        int all_configured = 1;
        for (int i = 0; i < monitor_count; i++) {
            if (!monitors[i].configured) {
                all_configured = 0;
                break;
            }
        }
        if (all_configured) break;
        wl_display_dispatch(display);
    }
    
    // Create SHM buffers for each monitor
    for (int i = 0; i < monitor_count; i++) {
        MonitorSurface *m = &monitors[i];
        m->shm_size = (size_t)m->width * m->height * 4;
        char name[64];
        snprintf(name, sizeof(name), "/wl_shm_%d_%d", getpid(), i);
        m->shm_fd = shm_open(name, O_CREAT | O_RDWR, 0600);
        shm_unlink(name);
        ftruncate(m->shm_fd, m->shm_size);
        m->shm_data = mmap(NULL, m->shm_size, PROT_READ | PROT_WRITE, MAP_SHARED, m->shm_fd, 0);
        memset(m->shm_data, 0, m->shm_size);
        
        struct wl_shm_pool *pool = wl_shm_create_pool(shm, m->shm_fd, m->shm_size);
        m->buffer = wl_shm_pool_create_buffer(pool, 0, m->width, m->height, m->width * 4, WL_SHM_FORMAT_XRGB8888);
        wl_shm_pool_destroy(pool);
    }
    
    return 0;
}

int c_layer_monitor_count(void) { return monitor_count; }
int c_layer_monitor_width(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].width : 0; }
int c_layer_monitor_height(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].height : 0; }
int c_layer_monitor_x(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].x : 0; }
int c_layer_monitor_y(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].y : 0; }
int c_layer_monitor_scale(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].scale : 1; }
const char* c_layer_monitor_name(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].name : ""; }
unsigned char* c_layer_monitor_buffer(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].shm_data : NULL; }
size_t c_layer_monitor_size(int idx) { return (idx >= 0 && idx < monitor_count) ? monitors[idx].shm_size : 0; }

// Get SHM buffer pointer for direct Go writes (no copy needed)
// Returns: pointer, size, width, height for monitor idx
int c_layer_monitor_get_buffer_info(int idx, unsigned char **ptr, size_t *size, int *w, int *h) {
    if (idx < 0 || idx >= monitor_count) return -1;
    MonitorSurface *m = &monitors[idx];
    *ptr = m->shm_data;
    *size = m->shm_size;
    *w = m->width;
    *h = m->height;
    return 0;
}

// Commit all monitors (no render, just attach + flush)
int c_layer_commit_all(void) {
    for (int i = 0; i < monitor_count; i++) {
        MonitorSurface *m = &monitors[i];
        if (!m->surface || !m->buffer) continue;
        wl_surface_attach(m->surface, m->buffer, 0, 0);
        wl_surface_damage(m->surface, 0, 0, m->width, m->height);
        wl_surface_commit(m->surface);
    }
    wl_display_flush(display);
    wl_display_dispatch_pending(display);
    return running ? 0 : -1;
}

void* c_layer_wl_display(void) { return (void*)display; }
void* c_layer_wl_surface(int idx) { return (idx >= 0 && idx < monitor_count) ? (void*)monitors[idx].surface : NULL; }

void c_layer_cleanup(void) {
    for (int i = 0; i < monitor_count; i++) {
        MonitorSurface *m = &monitors[i];
        if (m->buffer) wl_buffer_destroy(m->buffer);
        if (m->shm_data) munmap(m->shm_data, m->shm_size);
        if (m->shm_fd >= 0) close(m->shm_fd);
        if (m->layer_surface) zwlr_layer_surface_v1_destroy(m->layer_surface);
        if (m->surface) wl_surface_destroy(m->surface);
    }
    if (display) wl_display_disconnect(display);
    monitor_count = 0;
}
*/
import "C"
import (
	"context"
	"errors"
	"image"
	"sync"
	"unsafe"
)

type LayerSurface struct {
	mu         sync.RWMutex
	state      SurfaceState
	outputs    []Output
	bounds     image.Rectangle
	// Per-monitor SHM buffers (direct Go access, no copy)
	monitors   []MonitorBuffer
}

type MonitorBuffer struct {
	ptr    *byte
	size   uintptr
	width  int
	height int
}

func NewLayerSurface() *LayerSurface {
	return &LayerSurface{state: SurfaceStateInitialized}
}

func (s *LayerSurface) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret := C.c_layer_init()
	if ret == -1 {
		return errors.New("failed to connect to wayland display")
	}
	if ret == -2 {
		return errors.New("missing required Wayland protocols (compositor/shm/layer-shell)")
	}
	if ret == -3 {
		return errors.New("no outputs found")
	}

	count := int(C.c_layer_monitor_count())
	s.outputs = make([]Output, count)
	s.monitors = make([]MonitorBuffer, count)
	maxWidth := 0
	maxHeight := 0
	for i := 0; i < count; i++ {
		w := int(C.c_layer_monitor_width(C.int(i)))
		h := int(C.c_layer_monitor_height(C.int(i)))
		s.outputs[i] = Output{
			Name:   C.GoString(C.c_layer_monitor_name(C.int(i))),
			Width:  w,
			Height: h,
			Scale:  int(C.c_layer_monitor_scale(C.int(i))),
		}
		if w > maxWidth {
			maxWidth = w
		}
		if h > maxHeight {
			maxHeight = h
		}

		// Get direct pointer to SHM buffer
		var ptr *C.uchar
		var size C.size_t
		var cw, ch C.int
		C.c_layer_monitor_get_buffer_info(C.int(i), &ptr, &size, &cw, &ch)
		s.monitors[i] = MonitorBuffer{
			ptr:    (*byte)(unsafe.Pointer(ptr)),
			size:   uintptr(size),
			width:  int(cw),
			height: int(ch),
		}
	}

	s.bounds = image.Rect(0, 0, maxWidth, maxHeight)
	s.state = SurfaceStateRunning
	return nil
}

func (s *LayerSurface) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == SurfaceStateStopped {
		return nil
	}
	C.c_layer_cleanup()
	s.state = SurfaceStateStopped
	return nil
}

func (s *LayerSurface) CreateSurface(output Output) error {
	return nil // Surfaces created in Connect
}

func (s *LayerSurface) Outputs() []Output {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.outputs
}

func (s *LayerSurface) Buffer() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.monitors) == 0 {
		return nil
	}
	m := &s.monitors[0]
	return (*[1 << 30]byte)(unsafe.Pointer(m.ptr))[:m.size:m.size]
}

func (s *LayerSurface) Bounds() image.Rectangle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bounds
}

func (s *LayerSurface) Commit() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Copy monitor 0's buffer to all other monitors
	if len(s.monitors) > 1 {
		src := (*[1 << 30]byte)(unsafe.Pointer(s.monitors[0].ptr))[:s.monitors[0].size:s.monitors[0].size]
		for i := 1; i < len(s.monitors); i++ {
			dst := (*[1 << 30]byte)(unsafe.Pointer(s.monitors[i].ptr))[:s.monitors[i].size:s.monitors[i].size]
			copyLen := len(src)
			if len(dst) < copyLen {
				copyLen = len(dst)
			}
			copy(dst, src)
		}
	}

	ret := C.c_layer_commit_all()
	if ret != 0 {
		return errors.New("failed to commit")
	}
	return nil
}

func (s *LayerSurface) State() SurfaceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *LayerSurface) SetFrameCallback(fn func()) {}

// WlDisplayPtr returns the raw wl_display pointer for EGL integration.
func (s *LayerSurface) WlDisplayPtr() unsafe.Pointer {
	return unsafe.Pointer(C.c_layer_wl_display())
}

// WlSurfacePtr returns the raw wl_surface pointer for the given monitor index.
func (s *LayerSurface) WlSurfacePtr(idx int) unsafe.Pointer {
	return unsafe.Pointer(C.c_layer_wl_surface(C.int(idx)))
}

var _ Surface = (*LayerSurface)(nil)
