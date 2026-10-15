// The library is whatever is on disk under one folder right now: no index, no
// database. A rescan is a fresh walk, which on a warm cache is seconds even
// for tens of thousands of files.

#include "optics.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

const char* const kKindName[K_COUNT] = { "STL", "OBJ", "FBX", "SCAD", "3MF", "STEP" };

wchar_t* Widen(const char* s) {
    int n = MultiByteToWideChar(CP_UTF8, 0, s, -1, NULL, 0);
    wchar_t* w = n ? malloc(n * sizeof(wchar_t)) : NULL;
    if (w) MultiByteToWideChar(CP_UTF8, 0, s, -1, w, n);
    return w;
}

char* Narrow(const wchar_t* w) {
    int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL);
    char* s = n ? malloc(n) : NULL;
    if (s) WideCharToMultiByte(CP_UTF8, 0, w, -1, s, n, NULL, NULL);
    return s;
}

Bytes ReadWholeFile(const char* path) {
    Bytes b = { 0 };
    wchar_t* w = Widen(path);
    HANDLE h = w ? CreateFileW(w, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, 0, NULL)
                 : INVALID_HANDLE_VALUE;
    free(w);
    if (h == INVALID_HANDLE_VALUE) return b;
    LARGE_INTEGER size;
    // ponytail: whole-file reads, capped at 2 GB; stream if models get bigger.
    if (GetFileSizeEx(h, &size) && size.QuadPart < 0x7FFFFFFF) {
        DWORD got = 0;
        b.data = malloc((size_t)size.QuadPart + 1);
        if (b.data && ReadFile(h, b.data, (DWORD)size.QuadPart, &got, NULL) && got == size.QuadPart) {
            b.len = got;
        } else {
            free(b.data);
            b.data = NULL;
        }
    }
    CloseHandle(h);
    return b;
}

BOOL KindOf(const char* name, Kind* out) {
    static const struct { const char* ext; Kind k; } map[] = {
        { ".stl", K_STL }, { ".obj", K_OBJ }, { ".fbx", K_FBX }, { ".scad", K_SCAD },
        { ".3mf", K_3MF }, { ".step", K_STEP }, { ".stp", K_STEP },
    };
    const char* dot = strrchr(name, '.');
    if (!dot) return FALSE;
    for (size_t i = 0; i < sizeof(map) / sizeof(map[0]); i++)
        if (_stricmp(dot, map[i].ext) == 0) { *out = map[i].k; return TRUE; }
    return FALSE;
}

static char* Dup(const char* s) { return _strdup(s); }

static void Add(Library* lib, const char* path, const char* inner, const char* rel,
                uint64_t size, Kind kind) {
    if (lib->count == lib->cap) {
        int cap = lib->cap ? lib->cap * 2 : 256;
        Item* grown = realloc(lib->items, cap * sizeof(Item));
        if (!grown) return;
        lib->items = grown;
        lib->cap = cap;
    }
    Item* it = &lib->items[lib->count];
    it->path = Dup(path);
    it->inner = inner ? Dup(inner) : NULL;
    it->rel = Dup(rel);
    const char* base = strrchr(inner ? inner : rel, '/');
    it->name = Dup(base ? base + 1 : (inner ? inner : rel));
    it->size = size;
    it->kind = kind;
    lib->count++;
    InterlockedIncrement(&lib->found);
}

static void AddZip(Library* lib, const char* path, const char* rel) {
    ZipIndex z;
    if (!Zip_List(path, &z)) return;
    for (int i = 0; i < z.count; i++) {
        Kind k;
        if (!KindOf(z.entries[i].name, &k)) continue;
        size_t n = strlen(rel) + strlen(z.entries[i].name) + 2;
        char* r = malloc(n);
        if (!r) continue;
        sprintf_s(r, n, "%s!%s", rel, z.entries[i].name);
        Add(lib, path, z.entries[i].name, r, z.entries[i].usize, k);
        free(r);
    }
    Zip_Free(&z);
}

// dir and rel carry a trailing separator when not empty.
static void Walk(Library* lib, const wchar_t* dir, const char* rel) {
    size_t dlen = wcslen(dir);
    wchar_t* pattern = malloc((dlen + 2) * sizeof(wchar_t));
    if (!pattern) return;
    swprintf(pattern, dlen + 2, L"%s*", dir);
    WIN32_FIND_DATAW fd;
    HANDLE h = FindFirstFileExW(pattern, FindExInfoBasic, &fd, FindExSearchNameMatch, NULL,
                                FIND_FIRST_EX_LARGE_FETCH);
    free(pattern);
    if (h == INVALID_HANDLE_VALUE) return;
    do {
        if (lib->cancel) break;
        if (wcscmp(fd.cFileName, L".") == 0 || wcscmp(fd.cFileName, L"..") == 0) continue;
        // Junctions and symlinks are not followed: a loop would never end.
        if (fd.dwFileAttributes & FILE_ATTRIBUTE_REPARSE_POINT) continue;
        char* name = Narrow(fd.cFileName);
        if (!name) continue;
        size_t flen = dlen + wcslen(fd.cFileName) + 2;
        wchar_t* full = malloc(flen * sizeof(wchar_t));
        size_t rlen = strlen(rel) + strlen(name) + 2;
        char* r = malloc(rlen);
        if (full && r) {
            sprintf_s(r, rlen, "%s%s", rel, name);
            if (fd.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) {
                swprintf(full, flen, L"%s%s\\", dir, fd.cFileName);
                strcat_s(r, rlen, "/");
                Walk(lib, full, r);
            } else {
                swprintf(full, flen, L"%s%s", dir, fd.cFileName);
                uint64_t size = ((uint64_t)fd.nFileSizeHigh << 32) | fd.nFileSizeLow;
                const char* dot = strrchr(name, '.');
                Kind k;
                char* path = Narrow(full);
                if (path && dot && _stricmp(dot, ".zip") == 0) AddZip(lib, path, r);
                else if (path && KindOf(name, &k)) Add(lib, path, NULL, r, size, k);
                free(path);
            }
        }
        free(full);
        free(r);
        free(name);
    } while (FindNextFileW(h, &fd));
    FindClose(h);
}

void Scan(const char* root, Library* lib) {
    wchar_t* w = Widen(root);
    if (!w) return;
    size_t n = wcslen(w);
    wchar_t* dir = malloc((n + 2) * sizeof(wchar_t));
    if (dir) {
        wcscpy_s(dir, n + 2, w);
        if (n && dir[n - 1] != L'\\' && dir[n - 1] != L'/') wcscat_s(dir, n + 2, L"\\");
        Walk(lib, dir, "");
    }
    free(dir);
    free(w);
}

void Library_Free(Library* lib) {
    for (int i = 0; i < lib->count; i++) {
        Item* it = &lib->items[i];
        free(it->path); free(it->inner); free(it->name); free(it->rel);
    }
    free(lib->items);
    memset(lib, 0, sizeof(*lib));
}
