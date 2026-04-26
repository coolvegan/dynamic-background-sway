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

void c_layer_commit_surface(int idx);

static EGLDisplay g_egl_display = EGL_NO_DISPLAY;
static EGLContext g_egl_context = EGL_NO_CONTEXT;
static EGLSurface g_egl_surface = EGL_NO_SURFACE;
static struct wl_egl_window *g_egl_window = NULL;
static GLuint g_shader_program = 0;
static GLuint g_texture_shader = 0;
static GLuint g_texture_id = 0;
static int g_width = 0;
static int g_height = 0;

static const char* g_vs_color =
    "attribute vec2 a_pos;\n"
    "void main() { gl_Position = vec4(a_pos, 0.0, 1.0); }\n";

static const char* g_fs_color =
    "precision mediump float;\n"
    "uniform vec4 u_color;\n"
    "void main() { gl_FragColor = u_color; }\n";

static const char* g_vs_tex =
    "attribute vec2 a_pos;\n"
    "varying vec2 v_uv;\n"
    "void main() {\n"
    "    v_uv = a_pos * 0.5 + 0.5;\n"
    "    v_uv.y = 1.0 - v_uv.y;\n"
    "    gl_Position = vec4(a_pos, 0.0, 1.0);\n"
    "}\n";

static const char* g_fs_tex =
    "precision mediump float;\n"
    "varying vec2 v_uv;\n"
    "uniform sampler2D u_tex;\n"
    "void main() { gl_FragColor = texture2D(u_tex, v_uv); }\n";

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

static GLuint link_program(GLuint vs, GLuint fs) {
    GLuint p = glCreateProgram();
    glAttachShader(p, vs);
    glAttachShader(p, fs);
    glLinkProgram(p);
    GLint ok;
    glGetProgramiv(p, GL_LINK_STATUS, &ok);
    if (!ok) {
        char log[256];
        glGetProgramInfoLog(p, sizeof(log), NULL, log);
        fprintf(stderr, "shader link error: %s\n", log);
        return 0;
    }
    return p;
}

int egl_init(void* wl_display_ptr, void* wl_surface_ptr, int width, int height) {
    struct wl_display* wl_display = (struct wl_display*)wl_display_ptr;
    struct wl_surface* wl_surface = (struct wl_surface*)wl_surface_ptr;
    g_width = width;
    g_height = height;

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

    GLuint vs1 = compile_shader(GL_VERTEX_SHADER, g_vs_color);
    GLuint fs1 = compile_shader(GL_FRAGMENT_SHADER, g_fs_color);
    if (!vs1 || !fs1) return -8;
    g_shader_program = link_program(vs1, fs1);
    if (!g_shader_program) return -8;

    GLuint vs2 = compile_shader(GL_VERTEX_SHADER, g_vs_tex);
    GLuint fs2 = compile_shader(GL_FRAGMENT_SHADER, g_fs_tex);
    if (!vs2 || !fs2) return -9;
    g_texture_shader = link_program(vs2, fs2);
    if (!g_texture_shader) return -9;

    glGenTextures(1, &g_texture_id);
    glBindTexture(GL_TEXTURE_2D, g_texture_id);
    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, width, height, 0, GL_RGBA, GL_UNSIGNED_BYTE, NULL);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);

    glViewport(0, 0, width, height);
    glClearColor(0, 0, 0, 1);

    fprintf(stderr, "egl: ready %dx%d\n", width, height);
    return 0;
}

int egl_render_frame(unsigned char* pixels) {
    glBindTexture(GL_TEXTURE_2D, g_texture_id);
    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, g_width, g_height, 0, GL_RGBA, GL_UNSIGNED_BYTE, pixels);

    GLenum err = glGetError();
    if (err != GL_NO_ERROR) {
        fprintf(stderr, "egl: gl error after tex upload: 0x%x\n", err);
    }

    glClear(GL_COLOR_BUFFER_BIT);

    glUseProgram(g_texture_shader);
    GLint loc = glGetUniformLocation(g_texture_shader, "u_tex");
    glUniform1i(loc, 0);
    glActiveTexture(GL_TEXTURE0);
    glBindTexture(GL_TEXTURE_2D, g_texture_id);

    GLfloat verts[] = {
        -1.0f, -1.0f,
         1.0f, -1.0f,
        -1.0f,  1.0f,
         1.0f,  1.0f,
    };
    GLint pos = glGetAttribLocation(g_texture_shader, "a_pos");
    glEnableVertexAttribArray(pos);
    glVertexAttribPointer(pos, 2, GL_FLOAT, GL_FALSE, 0, verts);
    glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);

    err = glGetError();
    if (err != GL_NO_ERROR) {
        fprintf(stderr, "egl: gl error after draw: 0x%x\n", err);
    }

    fprintf(stderr, "egl: swapbuffers\n");
    eglSwapBuffers(g_egl_display, g_egl_surface);
    c_layer_commit_surface(0);
    return 0;
}

void egl_cleanup(void) {
    if (g_texture_id) {
        glDeleteTextures(1, &g_texture_id);
        g_texture_id = 0;
    }
    if (g_shader_program) {
        glDeleteProgram(g_shader_program);
        g_shader_program = 0;
    }
    if (g_texture_shader) {
        glDeleteProgram(g_texture_shader);
        g_texture_shader = 0;
    }
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
	"image"
	"image/color"
	"os"
	"unsafe"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/wayland"
)

type EGLRenderer struct {
	eglSurface  wayland.EGLSurfaceProvider
	config      domain.BackgroundConfig
	widgets     []*domain.Widget
	width       int
	height      int
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
	fmt.Fprintf(os.Stderr, "egl: Init() called, eglSurface=%v, initialized=%v\n", r.eglSurface != nil, r.initialized)
	if r.eglSurface == nil {
		return fmt.Errorf("EGLRenderer.Init: surface does not implement EGLSurfaceProvider")
	}
	if r.initialized {
		return nil
	}

	displayPtr := r.eglSurface.WlDisplayPtr()
	surfacePtr := r.eglSurface.WlSurfacePtr(0)
	fmt.Fprintf(os.Stderr, "egl: pointers display=%v surface=%v size=%dx%d\n", displayPtr, surfacePtr, r.width, r.height)
	if displayPtr == nil || surfacePtr == nil {
		return fmt.Errorf("EGLRenderer.Init: null Wayland pointers (display=%v, surface=%v)", displayPtr, surfacePtr)
	}

	fmt.Fprintf(os.Stderr, "egl: calling C.egl_init with %dx%d\n", r.width, r.height)
	ret := C.egl_init(displayPtr, surfacePtr, C.int(r.width), C.int(r.height))
	fmt.Fprintf(os.Stderr, "egl: C.egl_init returned %d\n", ret)
	if ret != 0 {
		return fmt.Errorf("egl init failed: %d", ret)
	}
	r.initialized = true
	fmt.Fprintf(os.Stderr, "egl: initialization complete\n")
	return nil
}

func (r *EGLRenderer) Render(ctx context.Context, rc *RenderContext) error {
	if !r.initialized {
		if err := r.Init(); err != nil {
			return err
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))

	if err := drawBackground(img, r.config); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}

	widgets := rc.Widgets()
	for _, w := range widgets {
		drawWidget(img, w)
	}

	pixels := img.Pix
	ret := C.egl_render_frame((*C.uchar)(unsafe.Pointer(&pixels[0])))
	if ret != 0 {
		return fmt.Errorf("egl render failed: %d", ret)
	}
	return nil
}

func (r *EGLRenderer) Clear() error {
	if !r.initialized {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))
	drawSolidBackground(img, color.RGBA{0, 0, 0, 255})
	pixels := img.Pix
	ret := C.egl_render_frame((*C.uchar)(unsafe.Pointer(&pixels[0])))
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
