#ifndef OPTICS_H
#define OPTICS_H

// Shared types for the core library (scan, zip, model, render) used by both
// Optics.exe (the window) and optics-check.exe (the console harness).
// Strings are UTF-8 throughout; they become UTF-16 only at Win32 calls.
// Design: docs/architecture.md.

#include <windows.h>
#include <stdint.h>
#include <stddef.h>

// ---- utf8 <-> utf16 ------------------------------------------------------
wchar_t* Widen(const char* s);      // caller frees
char* Narrow(const wchar_t* s);     // caller frees

// ---- bytes ---------------------------------------------------------------
typedef struct {
    unsigned char* data;
    size_t len;
} Bytes;

Bytes ReadWholeFile(const char* path);   // data NULL on failure; free(data)

// ---- zip -----------------------------------------------------------------
// Names inside an archive are listed, and one entry is read at a time with a
// seek, so a multi-GB zip costs its central directory, not its size.
// ponytail: no zip64 and no encryption; such entries are skipped, add zip64
// when someone has a >4 GB archive.
typedef struct {
    char* name;
    uint32_t method, crc;
    uint64_t csize, usize, offset;
} ZipEntry;

typedef struct {
    ZipEntry* entries;
    int count;
} ZipIndex;

BOOL Zip_List(const char* path, ZipIndex* out);
Bytes Zip_Read(const char* path, const ZipEntry* e);
void Zip_Free(ZipIndex* z);

// ---- scan ----------------------------------------------------------------
typedef enum { K_STL, K_OBJ, K_FBX, K_SCAD, K_3MF, K_STEP, K_COUNT } Kind;
extern const char* const kKindName[K_COUNT];

typedef struct {
    char* path;     // file on disk (the .zip itself for archive members)
    char* inner;    // entry inside the zip, or NULL
    char* name;     // file name shown in the list
    char* rel;      // path under the root, "a/b.zip!inner/c.stl" for members
    uint64_t size;
    Kind kind;
} Item;

typedef struct {
    Item* items;
    int count, cap;
    volatile LONG found;  // progress, readable from another thread
    volatile LONG cancel;
} Library;

void Scan(const char* root, Library* lib);
void Library_Free(Library* lib);
BOOL KindOf(const char* name, Kind* out);

// ---- model ---------------------------------------------------------------
// A loaded mesh, flattened to non-indexed triangles in the layout of
// GL_T2F_N3F_V3F so the renderer can hand it to glInterleavedArrays as is.
typedef struct {
    unsigned char* rgba;     // top row first
    int w, h;
    unsigned int gltex;      // set by the renderer
} Image;

typedef struct {
    float* verts;            // 8 floats per vertex: u v nx ny nz x y z
    int nverts;
    int image;               // index into Model.images, or -1
    float color[4];          // material colour, multiplied with the texture
} Part;

typedef struct {
    Part* parts;
    int nparts;
    Image* images;
    int nimages;
    float lo[3], hi[3];      // bounds
    long long triangles;
    int tex_wanted, tex_loaded;
    char missing[256];       // first missing texture names, for the status bar
    char error[256];
} Model;

// Loads STL/OBJ/FBX from disk or from inside a zip. Returns FALSE with
// m->error set when nothing could be read. A model whose textures are
// missing still loads; tex_wanted/tex_loaded say what happened.
BOOL Model_Load(const Item* it, Model* m);
void Model_Free(Model* m);

// ---- render --------------------------------------------------------------
typedef struct {
    float yaw, pitch, zoom;  // degrees, degrees, 1 = fit
} Camera;

// Needs a current GL context. Uploads textures on first draw.
void Render_Draw(Model* m, const Camera* cam, int w, int h);
void Render_Forget(Model* m);   // drop GL textures before Model_Free
// Draws offscreen with the software GL that renders into a DIB, so it works
// with no window, no display (CI, RDP) and on any thread. Returns top-down
// RGBA, w*h*4 bytes, caller frees; NULL on failure.
unsigned char* Render_Offscreen(Model* m, const Camera* cam, int w, int h);
// Render_Offscreen, written as a PNG.
BOOL Render_ToPng(Model* m, const Camera* cam, int w, int h, const char* png);

#endif // OPTICS_H
