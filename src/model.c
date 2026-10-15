// Mesh loading. STL is read here (the format is a page long); FBX and OBJ go
// through ufbx, which also hands back each material's texture reference.
//
// The hard part is not parsing, it is finding the texture files: an FBX
// carries the path the artist's machine used, which almost never exists on
// this one. Related() is the search order; docs/textures.md says why each
// step is there.

#include "optics.h"
#include "ufbx.h"

#define STB_IMAGE_IMPLEMENTATION
#define STBI_NO_HDR
#define STBI_NO_LINEAR
#include "stb_image.h"

#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ---------------------------------------------------------------------------
// Growing arrays and bounds
// ---------------------------------------------------------------------------

static BOOL Grow(void** p, int* cap, int need, size_t size) {
    if (need <= *cap) return TRUE;
    int cap2 = *cap ? *cap : 64;
    while (cap2 < need) cap2 *= 2;
    void* q = realloc(*p, cap2 * size);
    if (!q) return FALSE;
    *p = q;
    *cap = cap2;
    return TRUE;
}

static Part* NewPart(Model* m, int* cap) {
    if (!Grow((void**)&m->parts, cap, m->nparts + 1, sizeof(Part))) return NULL;
    Part* p = &m->parts[m->nparts++];
    memset(p, 0, sizeof(*p));
    p->image = -1;
    p->color[0] = p->color[1] = p->color[2] = 0.8f;
    p->color[3] = 1;
    return p;
}

static void Bounds(Model* m) {
    for (int k = 0; k < 3; k++) { m->lo[k] = 1e30f; m->hi[k] = -1e30f; }
    for (int i = 0; i < m->nparts; i++)
        for (int v = 0; v < m->parts[i].nverts; v++)
            for (int k = 0; k < 3; k++) {
                float x = m->parts[i].verts[v * 8 + 5 + k];
                if (x < m->lo[k]) m->lo[k] = x;
                if (x > m->hi[k]) m->hi[k] = x;
            }
    if (m->lo[0] > m->hi[0])
        for (int k = 0; k < 3; k++) m->lo[k] = m->hi[k] = 0;
}

// ---------------------------------------------------------------------------
// STL
// ---------------------------------------------------------------------------

// STL is Z-up by convention (it comes from printers); the viewer is Y-up.
static void PutTri(float* out, const float p[3][3]) {
    float a[3], b[3], n[3];
    for (int k = 0; k < 3; k++) { a[k] = p[1][k] - p[0][k]; b[k] = p[2][k] - p[0][k]; }
    n[0] = a[1] * b[2] - a[2] * b[1];
    n[1] = a[2] * b[0] - a[0] * b[2];
    n[2] = a[0] * b[1] - a[1] * b[0];
    float len = sqrtf(n[0] * n[0] + n[1] * n[1] + n[2] * n[2]);
    if (len > 0) for (int k = 0; k < 3; k++) n[k] /= len;
    for (int v = 0; v < 3; v++) {
        float* o = out + v * 8;
        o[0] = o[1] = 0;
        o[2] = n[0]; o[3] = n[2]; o[4] = -n[1];
        o[5] = p[v][0]; o[6] = p[v][2]; o[7] = -p[v][1];
    }
}

static BOOL LoadStl(const Bytes* b, Model* m) {
    int cap = 0;
    Part* part = NewPart(m, &cap);
    if (!part) return FALSE;
    // Binary first: plenty of binary STLs start their header with "solid".
    uint32_t n = b->len >= 84 ? *(const uint32_t*)(b->data + 80) : 0;
    if (b->len >= 84 && (uint64_t)84 + (uint64_t)n * 50 == b->len) {
        part->verts = malloc((size_t)n * 24 * sizeof(float) + 1);
        if (!part->verts) return FALSE;
        for (uint32_t t = 0; t < n; t++) {
            float p[3][3];
            memcpy(p, b->data + 84 + (size_t)t * 50 + 12, sizeof(p));
            PutTri(part->verts + (size_t)t * 24, p);
        }
        part->nverts = (int)n * 3;
    } else {
        // ASCII: every "vertex x y z", three to a facet; nothing else matters.
        int vcap = 0, count = 0;
        float p[3][3];
        char* text = malloc(b->len + 1);
        if (!text) return FALSE;
        memcpy(text, b->data, b->len);
        text[b->len] = 0;
        const char* s = text;
        while ((s = strstr(s, "vertex")) != NULL) {
            s += 6;
            char* e;
            for (int k = 0; k < 3; k++) { p[count % 3][k] = strtof(s, &e); s = e; }
            if (++count % 3 == 0) {
                if (!Grow((void**)&part->verts, &vcap, (count / 3) * 24, sizeof(float))) { count -= 3; break; }
                PutTri(part->verts + (count / 3 - 1) * 24, p);
            }
        }
        free(text);
        part->nverts = count / 3 * 3;
    }
    m->triangles = part->nverts / 3;
    return part->nverts > 0;
}

// ---------------------------------------------------------------------------
// Finding related files (textures, .mtl)
// ---------------------------------------------------------------------------

typedef struct {
    const Item* it;
    ZipIndex zip;   // members of the archive when the model is inside one
} Ctx;

static const char* BaseName(const char* s) {
    const char* b = s;
    for (const char* c = s; *c; c++) if (*c == '/' || *c == '\\') b = c + 1;
    return b;
}

static Bytes FromZip(Ctx* c, const char* want) {
    Bytes none = { 0 };
    for (int i = 0; i < c->zip.count; i++)
        if (_stricmp(c->zip.entries[i].name, want) == 0) return Zip_Read(c->it->path, &c->zip.entries[i]);
    return none;
}

static Bytes ByBaseFromZip(Ctx* c, const char* base) {
    Bytes none = { 0 };
    for (int i = 0; i < c->zip.count; i++)
        if (_stricmp(BaseName(c->zip.entries[i].name), base) == 0) return Zip_Read(c->it->path, &c->zip.entries[i]);
    return none;
}

// One candidate path: a member of the archive (by full path, or by name
// anywhere in it when dir is empty), or a file on disk.
static Bytes Get(Ctx* c, const char* dir, const char* path) {
    if (!c->it->inner) return ReadWholeFile(path);
    return dir[0] ? FromZip(c, path) : ByBaseFromZip(c, path);
}

// Same name, then the same stem with each image extension: textures get
// converted (a .tga in the FBX, a .png on disk) far more often than renamed.
static Bytes TryExts(Ctx* c, const char* dir, const char* base) {
    static const char* const exts[] = { ".png", ".jpg", ".jpeg", ".tga", ".bmp", ".psd" };
    char stem[MAX_PATH], path[MAX_PATH * 2];
    sprintf_s(path, sizeof(path), "%s%s", dir, base);
    Bytes b = Get(c, dir, path);
    strncpy_s(stem, sizeof(stem), base, _TRUNCATE);
    char* dot = strrchr(stem, '.');
    if (dot) *dot = 0;
    for (int i = 0; i < 6 && !b.data && dot; i++) {
        sprintf_s(path, sizeof(path), "%s%s%s", dir, stem, exts[i]);
        b = Get(c, dir, path);
    }
    return b;
}

static Bytes Related(Ctx* c, const char* name) {
    Bytes b = { 0 };
    if (!name || !name[0]) return b;
    char dir[MAX_PATH * 2], tmp[MAX_PATH * 2];
    const char* model = c->it->inner ? c->it->inner : c->it->path;
    strncpy_s(dir, sizeof(dir), model, _TRUNCATE);
    *(char*)BaseName(dir) = 0;  // keep the trailing separator
    const char* base = BaseName(name);
    BOOL absolute = name[0] == '/' || name[0] == '\\' || (name[0] && name[1] == ':');

    if (c->it->inner) {
        // Inside a zip: the stated relative path, then any member with that name.
        if (!absolute) {
            sprintf_s(tmp, sizeof(tmp), "%s%s", dir, name);
            for (char* p = tmp; *p; p++) if (*p == '\\') *p = '/';
            b = FromZip(c, tmp);
        }
        if (!b.data) b = TryExts(c, "", base);
        return b;
    }

    if (absolute) b = ReadWholeFile(name);
    if (!b.data && !absolute) {
        sprintf_s(tmp, sizeof(tmp), "%s%s", dir, name);
        b = ReadWholeFile(tmp);
    }
    // Next to the model, in textures/ beside it, and in textures/ one level
    // up: the layout Sketchfab and most asset stores ship (source/, textures/).
    static const char* const where[] = { "", "textures\\", "Textures\\", "..\\textures\\", "..\\Textures\\" };
    for (int i = 0; i < 5 && !b.data; i++) {
        sprintf_s(tmp, sizeof(tmp), "%s%s", dir, where[i]);
        b = TryExts(c, tmp, base);
    }
    return b;
}

// ufbx asks for an OBJ's .mtl through here, so it is found inside zips too.
static bool OpenRelated(void* user, ufbx_stream* stream, const char* path, size_t path_len,
                     const ufbx_open_file_info* info) {
    char name[MAX_PATH * 2];
    const ufbx_blob* raw = &info->original_filename;
    size_t n = raw->size ? raw->size : path_len;
    if (n >= sizeof(name)) return false;
    memcpy(name, raw->size ? raw->data : path, n);
    name[n] = 0;
    Bytes b = Related((Ctx*)user, name);
    if (!b.data) return false;
    bool ok = ufbx_open_memory(stream, b.data, b.len, NULL, NULL);  // copies
    free(b.data);
    return ok;
}

// ---------------------------------------------------------------------------
// FBX / OBJ
// ---------------------------------------------------------------------------

static void NoteMissing(Model* m, const char* name) {
    size_t used = strlen(m->missing);
    if (used + strlen(name) + 3 >= sizeof(m->missing)) return;
    sprintf_s(m->missing + used, sizeof(m->missing) - used, "%s%s", used ? ", " : "", name);
}

static int TextureFor(Ctx* c, Model* m, ufbx_texture* tex, ufbx_texture** seen, int* nseen,
                      int* cap) {
    if (!tex) return -1;
    if (tex->type != UFBX_TEXTURE_FILE && tex->file_textures.count) tex = tex->file_textures.data[0];
    for (int i = 0; i < *nseen; i++)
        if (seen[i] == tex) return m->images[i].rgba ? i : -1;
    if (*nseen >= 1024 || !Grow((void**)&m->images, cap, *nseen + 1, sizeof(Image))) return -1;

    Image* img = &m->images[*nseen];
    memset(img, 0, sizeof(*img));
    seen[(*nseen)++] = tex;
    m->nimages = *nseen;
    m->tex_wanted++;

    Bytes b = { 0 };
    if (tex->content.size) {
        img->rgba = stbi_load_from_memory(tex->content.data, (int)tex->content.size, &img->w, &img->h, NULL, 4);
    } else {
        const char* names[] = { tex->relative_filename.data, tex->filename.data, tex->absolute_filename.data };
        for (int i = 0; i < 3 && !b.data; i++) b = Related(c, names[i]);
        if (b.data) img->rgba = stbi_load_from_memory(b.data, (int)b.len, &img->w, &img->h, NULL, 4);
        free(b.data);
    }
    if (!img->rgba) {
        const char* n = tex->relative_filename.length ? tex->relative_filename.data : tex->filename.data;
        NoteMissing(m, BaseName(n));
        return -1;
    }
    m->tex_loaded++;
    return *nseen - 1;
}

static BOOL LoadUfbx(Ctx* c, const Bytes* b, Model* m) {
    ufbx_load_opts opts = { 0 };
    opts.target_axes = ufbx_axes_right_handed_y_up;
    opts.generate_missing_normals = true;
    opts.load_external_files = true;
    opts.ignore_missing_external_files = true;
    opts.filename.data = c->it->inner ? c->it->inner : c->it->path;
    opts.filename.length = strlen(opts.filename.data);
    opts.open_file_cb.fn = OpenRelated;
    opts.open_file_cb.user = c;

    ufbx_error err;
    ufbx_scene* scene = ufbx_load_memory(b->data, b->len, &opts, &err);
    if (!scene) {
        ufbx_format_error(m->error, sizeof(m->error), &err);
        char* nl = strchr(m->error, '\n');
        if (nl) *nl = 0;
        return FALSE;
    }

    int pcap = 0, icap = 0, nseen = 0;
    ufbx_texture* seen[1024];
    uint32_t* tri = NULL;
    size_t tricap = 0;
    stbi_set_flip_vertically_on_load(1);  // UV v=0 is the bottom row

    for (size_t ni = 0; ni < scene->nodes.count; ni++) {
        ufbx_node* node = scene->nodes.data[ni];
        ufbx_mesh* mesh = node->mesh;
        if (!mesh) continue;
        ufbx_matrix nmat = ufbx_matrix_for_normals(&node->geometry_to_world);
        if (tricap < mesh->max_face_triangles * 3) {
            tricap = mesh->max_face_triangles * 3;
            free(tri);
            tri = malloc(tricap * sizeof(uint32_t));
            if (!tri) break;
        }
        for (size_t pi = 0; pi < mesh->material_parts.count; pi++) {
            ufbx_mesh_part* mp = &mesh->material_parts.data[pi];
            if (!mp->num_triangles) continue;
            Part* part = NewPart(m, &pcap);
            if (!part) break;
            ufbx_material* mat = pi < node->materials.count ? node->materials.data[pi]
                               : pi < mesh->materials.count ? mesh->materials.data[pi] : NULL;
            if (mat) {
                ufbx_material_map* map = mat->pbr.base_color.texture ? &mat->pbr.base_color
                                                                      : &mat->fbx.diffuse_color;
                part->image = TextureFor(c, m, map->texture, seen, &nseen, &icap);
                if (part->image < 0 && mat->fbx.diffuse_color.has_value) {
                    ufbx_vec4 v = mat->fbx.diffuse_color.value_vec4;
                    part->color[0] = (float)v.x; part->color[1] = (float)v.y; part->color[2] = (float)v.z;
                }
                if (part->image >= 0) part->color[0] = part->color[1] = part->color[2] = 1;
            }
            part->verts = malloc(mp->num_triangles * 24 * sizeof(float));
            if (!part->verts) break;
            float* o = part->verts;
            for (size_t fi = 0; fi < mp->face_indices.count; fi++) {
                ufbx_face face = mesh->faces.data[mp->face_indices.data[fi]];
                uint32_t nt = ufbx_triangulate_face(tri, tricap, mesh, face);
                for (uint32_t k = 0; k < nt * 3; k++) {
                    uint32_t ix = tri[k];
                    ufbx_vec3 p = ufbx_transform_position(&node->geometry_to_world,
                                                          ufbx_get_vertex_vec3(&mesh->vertex_position, ix));
                    ufbx_vec3 n = mesh->vertex_normal.exists
                        ? ufbx_transform_direction(&nmat, ufbx_get_vertex_vec3(&mesh->vertex_normal, ix))
                        : (ufbx_vec3){ .y = 1 };
                    double len = sqrt(n.x * n.x + n.y * n.y + n.z * n.z);
                    if (len > 0) { n.x /= len; n.y /= len; n.z /= len; }
                    ufbx_vec2 uv = mesh->vertex_uv.exists ? ufbx_get_vertex_vec2(&mesh->vertex_uv, ix)
                                                          : (ufbx_vec2){ .x = 0 };
                    o[0] = (float)uv.x; o[1] = (float)uv.y;
                    o[2] = (float)n.x; o[3] = (float)n.y; o[4] = (float)n.z;
                    o[5] = (float)p.x; o[6] = (float)p.y; o[7] = (float)p.z;
                    o += 8;
                }
            }
            part->nverts = (int)((o - part->verts) / 8);
            m->triangles += part->nverts / 3;
        }
    }
    free(tri);
    ufbx_free_scene(scene);
    if (!m->triangles) strcpy_s(m->error, sizeof(m->error), "no meshes in the file");
    return m->triangles > 0;
}

// ---------------------------------------------------------------------------

BOOL Model_Load(const Item* it, Model* m) {
    memset(m, 0, sizeof(*m));
    Ctx c = { .it = it };
    Bytes b = { 0 };
    if (it->inner) {
        if (Zip_List(it->path, &c.zip))
            for (int i = 0; i < c.zip.count; i++)
                if (strcmp(c.zip.entries[i].name, it->inner) == 0) { b = Zip_Read(it->path, &c.zip.entries[i]); break; }
    } else {
        b = ReadWholeFile(it->path);
    }

    BOOL ok = FALSE;
    if (!b.data) strcpy_s(m->error, sizeof(m->error), "could not read the file");
    else if (it->kind == K_STL) {
        ok = LoadStl(&b, m);
        if (!ok) strcpy_s(m->error, sizeof(m->error), "no triangles in the STL");
    }
    else if (it->kind == K_FBX || it->kind == K_OBJ) ok = LoadUfbx(&c, &b, m);
    else strcpy_s(m->error, sizeof(m->error), "no 3D preview for this type");

    free(b.data);
    Zip_Free(&c.zip);
    if (ok) Bounds(m);
    else {
        char keep[256];
        strcpy_s(keep, sizeof(keep), m->error);
        Model_Free(m);
        strcpy_s(m->error, sizeof(m->error), keep);
    }
    return ok;
}

void Model_Free(Model* m) {
    for (int i = 0; i < m->nparts; i++) free(m->parts[i].verts);
    for (int i = 0; i < m->nimages; i++) stbi_image_free(m->images[i].rgba);
    free(m->parts);
    free(m->images);
    memset(m, 0, sizeof(*m));
}
