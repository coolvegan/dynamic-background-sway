//go:build cgo

package renderer

/*
#cgo pkg-config: egl glesv2 wayland-client wayland-egl
#include <EGL/egl.h>
#include <GLES2/gl2.h>
#include <wayland-client.h>
#include <wayland-egl.h>
#include <stdio.h>
#include <stdlib.h>

static EGLDisplay g_egl_display = EGL_NO_DISPLAY;
static EGLContext g_egl_context = EGL_NO_CONTEXT;
static EGLSurface g_egl_surface = EGL_NO_SURFACE;
static struct wl_egl_window *g_egl_window = NULL;
static GLuint g_shader_program = 0;
static GLint g_uniform_color = -1;

static const char* g_vs_source =
    "attribute vec2 a_pos;\n"
    "void main() { gl_Position = vec4(a_pos, 0.0, 1.0); }\n";

static const char* g_fs_source =
    "precision mediump float;\n"
    "uniform vec4 u_color;\n"
    "void main() { gl_FragColor = u_color; }\n";

static GLuint compile_shader(GLenum type, const char* src) {
    GLuint s = glCreateShader(type);
    glShaderSource(s, 1, &src, NULL);
    glCompileShader(s);
    GLint ok;
    glGetShaderiv(s, GL_COMPILE_STATUS, &ok);
    if (!ok) {
        char log[256];
        glGetShaderInfoLog(s, sizeof(log), NULL, log);
        fprintf(stderr, "shader compile error: %s\n", log);
        return 0;
    }
    return s;
}

int egl_init(void* wl_display_ptr, void* wl_surface_ptr, int width, int height) {
    struct wl_display* wl_display = (struct wl_display*)wl_display_ptr;
    struct wl_surface* wl_surface = (struct wl_surface*)wl_surface_ptr;
    
    g_egl_display = eglGetDisplay((EGLNativeDisplayType)wl_display);
    if (g_egl_display == EGL_NO_DISPLAY) {
        fprintf(stderr, "egl: no display\n");
        return -1;
    }
    
    EGLint major, minor;
    if (!eglInitialize(g_egl_display, &major, &minor)) {
        fprintf(stderr, "egl: init failed\n");
        return -2;
    }
    fprintf(stderr, "egl: initialized %d.%d\n", major, minor);
    
    EGLint attrs[] = {
        EGL_SURFACE_TYPE, EGL_WINDOW_BIT,
        EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT,
        EGL_RED_SIZE, 8, EGL_GREEN_SIZE, 8, EGL_BLUE_SIZE, 8, EGL_ALPHA_SIZE, 8,
        EGL_NONE
    };
    EGLConfig config;
    EGLint num_configs;
    if (!eglChooseConfig(g_egl_display, attrs, &config, 1, &num_configs)) {
        fprintf(stderr, "egl: choose config failed\n");
        return -3;
    }
    
    EGLint ctx_attrs[] = { EGL_CONTEXT_CLIENT_VERSION, 2, EGL_NONE };
    g_egl_context = eglCreateContext(g_egl_display, config, EGL_NO_CONTEXT, ctx_attrs);
    if (g_egl_context == EGL_NO_CONTEXT) {
        fprintf(stderr, "egl: create context failed\n");
        return -4;
    }
    
    g_egl_window = wl_egl_window_create(wl_surface, width, height);
    if (!g_egl_window) {
        fprintf(stderr, "egl: create window failed\n");
        return -5;
    }
    
    g_egl_surface = eglCreateWindowSurface(g_egl_display, config, (EGLNativeWindowType)g_egl_window, NULL);
    if (g_egl_surface == EGL_NO_SURFACE) {
        fprintf(stderr, "egl: create surface failed\n");
        return -6;
    }
    
    if (!eglMakeCurrent(g_egl_display, g_egl_surface, g_egl_surface, g_egl_context)) {
        fprintf(stderr, "egl: make current failed\n");
        return -7;
    }
    
    GLuint vs = compile_shader(GL_VERTEX_SHADER, g_vs_source);
    GLuint fs = compile_shader(GL_FRAGMENT_SHADER, g_fs_source);
    if (!vs || !fs) return -8;
    
    g_shader_program = glCreateProgram();
    glAttachShader(g_shader_program, vs);
    glAttachShader(g_shader_program, fs);
    glLinkProgram(g_shader_program);
    
    GLint link_ok;
    glGetProgramiv(g_shader_program, GL_LINK_STATUS, &link_ok);
    if (!link_ok) {
        char log[256];
        glGetProgramInfoLog(g_shader_program, sizeof(log), NULL, log);
        fprintf(stderr, "shader link error: %s\n", log);
        return -9;
    }
    
    glUseProgram(g_shader_program);
    g_uniform_color = glGetUniformLocation(g_shader_program, "u_color");
    glViewport(0, 0, width, height);
    glClearColor(0, 0, 0, 1);
    
    fprintf(stderr, "egl: ready %dx%d\n", width, height);
    return 0;
}

int egl_render_background(float r, float g, float b) {
    glClearColor(r, g, b, 1.0);
    glClear(GL_COLOR_BUFFER_BIT);
    eglSwapBuffers(g_egl_display, g_egl_surface);
    return 0;
}

void egl_cleanup(void) {
    if (g_egl_surface != EGL_NO_SURFACE) {
        eglDestroySurface(g_egl_display, g_egl_surface);
        g_egl_surface = EGL_NO_SURFACE;
    }
    if (g_egl_window) {
        wl_egl_window_destroy(g_egl_window);
        g_egl_window = NULL;
    }
    if (g_egl_context != EGL_NO_CONTEXT) {
        eglDestroyContext(g_egl_display, g_egl_context);
        g_egl_context = EGL_NO_CONTEXT;
    }
    if (g_egl_display != EGL_NO_DISPLAY) {
        eglTerminate(g_egl_display);
        g_egl_display = EGL_NO_DISPLAY;
    }
}
*/
import "C"
import (
	"context"
	"fmt"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/wayland"
)

type EGLRenderer struct {
	eglSurface wayland.EGLSurfaceProvider
	config     domain.BackgroundConfig
	width      int
	height     int
	initialized bool
}

func NewEGLRenderer(s wayland.Surface, config domain.BackgroundConfig) *EGLRenderer {
	bounds := s.Bounds()
	egl, ok := s.(wayland.EGLSurfaceProvider)
	if !ok {
		return &EGLRenderer{
			config: config,
			width:  bounds.Dx(),
			height: bounds.Dy(),
		}
	}
	return &EGLRenderer{
		eglSurface: egl,
		config:     config,
		width:      bounds.Dx(),
		height:     bounds.Dy(),
	}
}

func (r *EGLRenderer) Init() error {
	if r.eglSurface == nil {
		return fmt.Errorf("EGLRenderer.Init: surface does not implement EGLSurfaceProvider")
	}
	if r.initialized {
		return nil
	}

	displayPtr := r.eglSurface.WlDisplayPtr()
	surfacePtr := r.eglSurface.WlSurfacePtr(0)
	if displayPtr == nil || surfacePtr == nil {
		return fmt.Errorf("EGLRenderer.Init: null Wayland pointers")
	}

	ret := C.egl_init(displayPtr, surfacePtr, C.int(r.width), C.int(r.height))
	if ret != 0 {
		return fmt.Errorf("egl init failed: %d", ret)
	}
	r.initialized = true
	return nil
}

func (r *EGLRenderer) Render(ctx context.Context, rc *RenderContext) error {
	if !r.initialized {
		if err := r.Init(); err != nil {
			return err
		}
	}

	color := [3]float32{0, 0, 0}
	if r.config.Type == domain.BackgroundTypeSolid && len(r.config.Colors) > 0 {
		c, err := parseHexColor(r.config.Colors[0])
		if err == nil {
			color = [3]float32{float32(c.R) / 255.0, float32(c.G) / 255.0, float32(c.B) / 255.0}
		}
	}

	ret := C.egl_render_background(C.float(color[0]), C.float(color[1]), C.float(color[2]))
	if ret != 0 {
		return fmt.Errorf("egl render failed: %d", ret)
	}
	return nil
}

func (r *EGLRenderer) Clear() error {
	if !r.initialized {
		return nil
	}
	ret := C.egl_render_background(0, 0, 0)
	if ret != 0 {
		return fmt.Errorf("egl clear failed: %d", ret)
	}
	return nil
}

func (r *EGLRenderer) Cleanup() {
	if r.initialized {
		C.egl_cleanup()
		r.initialized = false
	}
}

func (r *EGLRenderer) SetConfig(cfg domain.BackgroundConfig) {
	r.config = cfg
}

func (r *EGLRenderer) PresentsOwnFrames() bool { return true }

var _ Renderer = (*EGLRenderer)(nil)
