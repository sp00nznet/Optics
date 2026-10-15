// Just enough of PKZIP to list an archive and pull one member out: the end
// record, the central directory, the local header's variable lengths. Stored
// and deflate are the only methods anything writes in practice.

#include "optics.h"
#include "inflate.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static uint32_t U16(const unsigned char* p) { return p[0] | p[1] << 8; }
static uint32_t U32(const unsigned char* p) { return p[0] | p[1] << 8 | p[2] << 16 | (uint32_t)p[3] << 24; }

static FILE* OpenRead(const char* path) {
    wchar_t* w = Widen(path);
    FILE* f = w ? _wfopen(w, L"rb") : NULL;
    free(w);
    return f;
}

BOOL Zip_List(const char* path, ZipIndex* out) {
    memset(out, 0, sizeof(*out));
    FILE* f = OpenRead(path);
    if (!f) return FALSE;

    // The end record sits in the last 22 bytes plus up to 64k of comment.
    _fseeki64(f, 0, SEEK_END);
    long long size = _ftelli64(f);
    long long tail = size < 22 + 65535 ? size : 22 + 65535;
    unsigned char* buf = malloc((size_t)tail);
    BOOL ok = FALSE;
    unsigned char* cd = NULL;
    if (!buf || tail < 22) goto done;
    _fseeki64(f, size - tail, SEEK_SET);
    if (fread(buf, 1, (size_t)tail, f) != (size_t)tail) goto done;

    const unsigned char* eocd = NULL;
    for (long long i = tail - 22; i >= 0; i--)
        if (U32(buf + i) == 0x06054b50) { eocd = buf + i; break; }
    if (!eocd) goto done;

    uint32_t count = U16(eocd + 10), cdsize = U32(eocd + 12), cdoff = U32(eocd + 16);
    if ((long long)cdoff + cdsize > size) goto done;
    cd = malloc(cdsize ? cdsize : 1);
    if (!cd) goto done;
    _fseeki64(f, cdoff, SEEK_SET);
    if (fread(cd, 1, cdsize, f) != cdsize) goto done;

    out->entries = calloc(count ? count : 1, sizeof(ZipEntry));
    if (!out->entries) goto done;
    size_t at = 0;
    for (uint32_t i = 0; i < count && at + 46 <= cdsize; i++) {
        const unsigned char* h = cd + at;
        if (U32(h) != 0x02014b50) break;
        uint32_t nlen = U16(h + 28), elen = U16(h + 30), clen = U16(h + 32);
        if (at + 46 + nlen > cdsize) break;
        at += 46 + nlen + elen + clen;
        uint32_t flags = U16(h + 8), method = U16(h + 10);
        uint32_t csize = U32(h + 20), usize = U32(h + 24);
        // Skip directories, encrypted members, zip64 sizes and unknown methods.
        if (nlen == 0 || h[46 + nlen - 1] == '/' || (flags & 1)) continue;
        if (csize == 0xFFFFFFFF || usize == 0xFFFFFFFF || (method != 0 && method != 8)) continue;
        ZipEntry* e = &out->entries[out->count++];
        e->name = malloc(nlen + 1);
        memcpy(e->name, h + 46, nlen);
        e->name[nlen] = 0;
        for (char* c = e->name; *c; c++) if (*c == '\\') *c = '/';
        e->method = method;
        e->crc = U32(h + 16);
        e->csize = csize;
        e->usize = usize;
        e->offset = U32(h + 42);
    }
    ok = TRUE;
done:
    free(buf);
    free(cd);
    fclose(f);
    if (!ok) Zip_Free(out);
    return ok;
}

Bytes Zip_Read(const char* path, const ZipEntry* e) {
    Bytes b = { 0 };
    FILE* f = OpenRead(path);
    if (!f) return b;
    unsigned char lh[30];
    unsigned char* packed = NULL;
    _fseeki64(f, (long long)e->offset, SEEK_SET);
    if (fread(lh, 1, 30, f) != 30 || U32(lh) != 0x04034b50) goto done;
    _fseeki64(f, (long long)e->offset + 30 + U16(lh + 26) + U16(lh + 28), SEEK_SET);
    packed = malloc((size_t)e->csize + 1);
    if (!packed || fread(packed, 1, (size_t)e->csize, f) != e->csize) goto done;
    if (e->method == 0) {
        b.data = packed;
        b.len = (size_t)e->csize;
        packed = NULL;
    } else {
        b.data = Inflate_Raw(packed, (size_t)e->csize, &b.len);
    }
    // The central directory's size is the one to trust; a mismatch is a
    // damaged archive, and a short mesh is worse than none.
    if (b.data && b.len != e->usize) {
        free(b.data);
        b.data = NULL;
        b.len = 0;
    }
done:
    free(packed);
    fclose(f);
    return b;
}

void Zip_Free(ZipIndex* z) {
    for (int i = 0; i < z->count; i++) free(z->entries[i].name);
    free(z->entries);
    memset(z, 0, sizeof(*z));
}
