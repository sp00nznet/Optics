// Drawing a Model with OpenGL 1.1 fixed function. 1.1 on purpose: it is what
// every Windows machine has with no loader, including Microsoft's software
// implementation over RDP and in a VM, and it is the one Render_ToPng uses to
// draw into a DIB with no window at all. A textured, lit mesh needs nothing
// newer. The cost is power-of-two textures and no mipmaps; see Upload().

#include "optics.h"

#include <GL/gl.h>
#include <math.h>
#include <stdlib.h>
#include <string.h>

#define STB_IMAGE_WRITE_IMPLEMENTATION
#define STBIW_WINDOWS_UTF8
#include "stb_image_write.h"

static int Pot(int x, int max) {
    int p = 1;
    while (p < x && p < max) p <<= 1;
    return p;
}

static void Upload(Image* img) {
    GLint max = 0;
    glGetIntegerv(GL_MAX_TEXTURE_SIZE, &max);
    if (max < 64) max = 1024;
    int w = Pot(img->w, max), h = Pot(img->h, max);
    unsigned char* px = img->rgba;
    unsigned char* tmp = NULL;
    // GL 1.1 only takes power-of-two sizes. ponytail: nearest-neighbour
    // resample, which is soft on odd-sized textures; use GL 2.0's NPOT path
    // if that ever matters.
    if (w != img->w || h != img->h) {
        tmp = malloc((size_t)w * h * 4);
        if (!tmp) return;
        for (int y = 0; y < h; y++)
            for (int x = 0; x < w; x++)
                memcpy(tmp + ((size_t)y * w + x) * 4,
                       img->rgba + ((size_t)(y * img->h / h) * img->w + x * img->w / w) * 4, 4);
        px = tmp;
    }
    glGenTextures(1, &img->gltex);
    glBindTexture(GL_TEXTURE_2D, img->gltex);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_REPEAT);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_REPEAT);
    glPixelStorei(GL_UNPACK_ALIGNMENT, 1);
    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, w, h, 0, GL_RGBA, GL_UNSIGNED_BYTE, px);
    free(tmp);
}

void Render_Draw(Model* m, const Camera* cam, int w, int h) {
    glViewport(0, 0, w, h);
    glClearColor(0.16f, 0.17f, 0.19f, 1);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    if (!m || !m->nparts || w <= 0 || h <= 0) return;

    float c[3], r = 0;
    for (int k = 0; k < 3; k++) {
        c[k] = (m->lo[k] + m->hi[k]) / 2;
        r += (m->hi[k] - m->lo[k]) * (m->hi[k] - m->lo[k]);
    }
    r = sqrtf(r) / 2;
    if (r <= 0) r = 1;
    const float fov = 0.6f;  // radians, vertical
    float dist = r / sinf(fov / 2) * cam->zoom;
    float zn = dist - r * 1.5f, zf = dist + r * 1.5f;
    if (zn < dist * 0.01f) zn = dist * 0.01f;
    float t = zn * tanf(fov / 2), aspect = (float)w / h;

    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    glFrustum(-t * aspect, t * aspect, -t, t, zn, zf);
    glMatrixMode(GL_MODELVIEW);
    glLoadIdentity();
    // A headlight, set before the camera moves so it moves with the viewer.
    static const GLfloat light[] = { 0.4f, 0.6f, 1, 0 }, ambient[] = { 0.35f, 0.35f, 0.35f, 1 };
    glLightfv(GL_LIGHT0, GL_POSITION, light);
    glLightModelfv(GL_LIGHT_MODEL_AMBIENT, ambient);
    glLightModeli(GL_LIGHT_MODEL_TWO_SIDE, GL_TRUE);  // plenty of meshes have flipped faces
    glTranslatef(0, 0, -dist);
    glRotatef(cam->pitch, 1, 0, 0);
    glRotatef(cam->yaw, 0, 1, 0);
    glTranslatef(-c[0], -c[1], -c[2]);

    glEnable(GL_DEPTH_TEST);
    glEnable(GL_LIGHTING);
    glEnable(GL_LIGHT0);
    glEnable(GL_COLOR_MATERIAL);
    glColorMaterial(GL_FRONT_AND_BACK, GL_AMBIENT_AND_DIFFUSE);
    glAlphaFunc(GL_GREATER, 0.5f);  // cut-out foliage and decals; no sorting needed
    glTexEnvi(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE);

    for (int i = 0; i < m->nparts; i++) {
        Part* p = &m->parts[i];
        if (p->image >= 0) {
            Image* img = &m->images[p->image];
            if (!img->gltex) Upload(img);
            glBindTexture(GL_TEXTURE_2D, img->gltex);
            glEnable(GL_TEXTURE_2D);
            glEnable(GL_ALPHA_TEST);
        } else {
            glDisable(GL_TEXTURE_2D);
            glDisable(GL_ALPHA_TEST);
        }
        glColor4fv(p->color);
        glInterleavedArrays(GL_T2F_N3F_V3F, 0, p->verts);
        glDrawArrays(GL_TRIANGLES, 0, p->nverts);
    }
    glDisable(GL_TEXTURE_2D);
    glDisable(GL_ALPHA_TEST);
}

void Render_Forget(Model* m) {
    for (int i = 0; i < m->nimages; i++)
        if (m->images[i].gltex) {
            glDeleteTextures(1, &m->images[i].gltex);
            m->images[i].gltex = 0;
        }
}

unsigned char* Render_Offscreen(Model* m, const Camera* cam, int w, int h) {
    unsigned char* out = NULL;
    HDC dc = CreateCompatibleDC(NULL);
    BITMAPINFO bi = { { sizeof(BITMAPINFOHEADER), w, h, 1, 32, BI_RGB } };
    void* bits = NULL;
    HBITMAP bmp = CreateDIBSection(dc, &bi, DIB_RGB_COLORS, &bits, NULL, 0);
    HGLRC rc = NULL;
    if (!dc || !bmp) goto done;
    SelectObject(dc, bmp);
    PIXELFORMATDESCRIPTOR pfd = { sizeof(pfd), 1, PFD_DRAW_TO_BITMAP | PFD_SUPPORT_OPENGL | PFD_SUPPORT_GDI,
                                  PFD_TYPE_RGBA, 32 };
    pfd.cDepthBits = 24;
    int pf = ChoosePixelFormat(dc, &pfd);
    if (!pf || !SetPixelFormat(dc, pf, &pfd)) goto done;
    rc = wglCreateContext(dc);
    if (!rc || !wglMakeCurrent(dc, rc)) goto done;

    Render_Draw(m, cam, w, h);
    glFinish();
    Render_Forget(m);

    // The DIB is bottom-up BGRX; callers get top-down RGBA.
    out = malloc((size_t)w * h * 4);
    if (out)
        for (int y = 0; y < h; y++)
            for (int x = 0; x < w; x++) {
                const unsigned char* s = (const unsigned char*)bits + ((size_t)(h - 1 - y) * w + x) * 4;
                unsigned char* d = out + ((size_t)y * w + x) * 4;
                d[0] = s[2]; d[1] = s[1]; d[2] = s[0]; d[3] = 255;
            }
done:
    if (rc) { wglMakeCurrent(NULL, NULL); wglDeleteContext(rc); }
    if (bmp) DeleteObject(bmp);
    if (dc) DeleteDC(dc);
    return out;
}

BOOL Render_ToPng(Model* m, const Camera* cam, int w, int h, const char* png) {
    unsigned char* px = Render_Offscreen(m, cam, w, h);
    BOOL ok = px && stbi_write_png(png, w, h, 4, px, w * 4) != 0;
    free(px);
    return ok;
}
