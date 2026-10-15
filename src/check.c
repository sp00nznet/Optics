// optics-check: the console side of Optics, for CI and for anyone without a
// display. A separate console exe rather than flags on Optics.exe because a
// GUI-subsystem program has no stdout and the shell does not wait for it.
//
//   optics-check --selftest
//   optics-check --check <folder> <expected.tsv> [--update]
//   optics-check --render <model> <out.png> [WxH]
//
// --check is the conformance harness (docs/conformance.md): every mesh under
// the folder is loaded and compared with the expected triangle and texture
// counts. Any difference, or an expected file that no longer loads, fails.

#include "optics.h"
#include "inflate.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static BOOL Previewable(Kind k) { return k == K_STL || k == K_OBJ || k == K_FBX; }

static int SelfTest(void) {
    int bad = 0;
    char why[256];
    if (!Inflate_SelfTest(why, sizeof(why))) { printf("FAIL inflate: %s\n", why); bad++; }
    else printf("ok   inflate\n");

    Kind k;
    BOOL kinds = KindOf("a.STL", &k) && k == K_STL && KindOf("b.fbx", &k) && k == K_FBX &&
                 KindOf("c.stp", &k) && k == K_STEP && !KindOf("d.txt", &k) && !KindOf("noext", &k);
    printf("%s kinds\n", kinds ? "ok  " : "FAIL");
    bad += !kinds;

    // An ASCII STL whose header also says "solid", the case binary detection
    // has to get right.
    char path[MAX_PATH], dir[MAX_PATH];
    GetTempPathA(MAX_PATH, dir);
    sprintf_s(path, sizeof(path), "%soptics-selftest.stl", dir);
    FILE* f = fopen(path, "wb");
    if (f) {
        fputs("solid t\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\n"
              "endloop\nendfacet\nendsolid t\n", f);
        fclose(f);
    }
    Item it = { path, NULL, "t.stl", "t.stl", 0, K_STL };
    Model m;
    BOOL stl = Model_Load(&it, &m) && m.triangles == 1 && m.hi[0] == 1 && m.hi[2] == 0 && m.lo[2] == -1;
    printf("%s ascii stl (z-up becomes y-up)\n", stl ? "ok  " : "FAIL");
    bad += !stl;
    Model_Free(&m);
    DeleteFileA(path);
    return bad ? 1 : 0;
}

typedef struct {
    char rel[512];
    long long tris;
    int loaded, wanted;
    BOOL seen;
} Expect;

static int Check(const char* root, const char* expected, BOOL update) {
    Expect* ex = NULL;
    int nex = 0, cap = 0;
    FILE* f = fopen(expected, "r");
    // Without its expected counts the harness can only report "new", which
    // would pass anything. Only --update may start from nothing.
    if (!f && !update) {
        printf("FAIL cannot read %s (create it with --update)\n", expected);
        return 1;
    }
    char line[700];
    while (f && fgets(line, sizeof(line), f)) {
        if (line[0] == '#' || line[0] == '\n') continue;
        if (nex == cap) {
            cap = cap ? cap * 2 : 64;
            ex = realloc(ex, cap * sizeof(Expect));
            if (!ex) return 2;
        }
        Expect* e = &ex[nex];
        memset(e, 0, sizeof(*e));
        if (sscanf_s(line, "%511[^\t]\t%lld\t%d/%d", e->rel, (unsigned)sizeof(e->rel), &e->tris,
                     &e->loaded, &e->wanted) == 4)
            nex++;
    }
    if (f) fclose(f);

    Library lib = { 0 };
    Scan(root, &lib);
    FILE* out = update ? fopen(expected, "w") : NULL;
    if (out) fprintf(out, "# path\ttriangles\ttextures loaded/wanted -- written by optics-check --update\n");

    int pass = 0, fail = 0, total = 0;
    for (int i = 0; i < lib.count; i++) {
        const Item* it = &lib.items[i];
        if (!Previewable(it->kind)) continue;
        Model m;
        BOOL ok = Model_Load(it, &m);
        Expect* e = NULL;
        for (int j = 0; j < nex; j++)
            if (strcmp(ex[j].rel, it->rel) == 0) { e = &ex[j]; break; }
        if (e) e->seen = TRUE;
        if (out && ok) fprintf(out, "%s\t%lld\t%d/%d\n", it->rel, m.triangles, m.tex_loaded, m.tex_wanted);

        BOOL good = ok && (!e || (e->tris == m.triangles && e->loaded == m.tex_loaded && e->wanted == m.tex_wanted));
        total++;
        if (good) pass++; else fail++;
        if (ok)
            printf("%s %s  tris=%lld tex=%d/%d%s%s%s\n", good ? "PASS" : "FAIL", it->rel, m.triangles,
                   m.tex_loaded, m.tex_wanted, e ? "" : "  (new)", m.missing[0] ? "  missing: " : "", m.missing);
        else
            printf("FAIL %s  %s\n", it->rel, m.error);
        if (e && !good && ok)
            printf("     expected tris=%lld tex=%d/%d\n", e->tris, e->loaded, e->wanted);
        Model_Free(&m);
    }
    for (int j = 0; j < nex; j++)
        if (!ex[j].seen && !update) {
            printf("FAIL %s  expected but not found\n", ex[j].rel);
            fail++;
            total++;
        }
    if (out) fclose(out);
    printf("\nconformance: %d/%d pass\n", pass, total);
    Library_Free(&lib);
    free(ex);
    return fail && !update ? 1 : 0;
}

static int RenderOne(const char* spec, const char* png, const char* size) {
    int w = 800, h = 600;
    if (size) sscanf_s(size, "%dx%d", &w, &h);
    char path[1024];
    strncpy_s(path, sizeof(path), spec, _TRUNCATE);
    char* bang = strchr(path, '!');  // archive.zip!inner/model.fbx
    if (bang) *bang = 0;
    Item it = { path, bang ? bang + 1 : NULL, path, path, 0, K_STL };
    if (!KindOf(bang ? bang + 1 : path, &it.kind)) {
        fprintf(stderr, "not a model file: %s\n", spec);
        return 2;
    }
    Model m;
    if (!Model_Load(&it, &m)) {
        fprintf(stderr, "%s: %s\n", spec, m.error);
        return 1;
    }
    Camera cam = { 35, 25, 1.05f };
    BOOL ok = Render_ToPng(&m, &cam, w, h, png);
    printf("%s: %lld triangles, textures %d/%d%s%s -> %s\n", spec, m.triangles, m.tex_loaded, m.tex_wanted,
           m.missing[0] ? ", missing " : "", m.missing, ok ? png : "render failed");
    Model_Free(&m);
    return ok ? 0 : 1;
}

int wmain(int argc, wchar_t** wargv) {
    char** argv = calloc(argc + 1, sizeof(char*));
    for (int i = 0; i < argc; i++) argv[i] = Narrow(wargv[i]);
    SetConsoleOutputCP(CP_UTF8);

    if (argc >= 2 && strcmp(argv[1], "--selftest") == 0) return SelfTest();
    if (argc >= 4 && strcmp(argv[1], "--check") == 0)
        return Check(argv[2], argv[3], argc >= 5 && strcmp(argv[4], "--update") == 0);
    if (argc >= 4 && strcmp(argv[1], "--render") == 0) return RenderOne(argv[2], argv[3], argc >= 5 ? argv[4] : NULL);

    fprintf(stderr,
            "usage:\n"
            "  optics-check --selftest\n"
            "  optics-check --check <folder> <expected.tsv> [--update]\n"
            "  optics-check --render <model[.zip!member]> <out.png> [WxH]\n");
    return 2;
}
