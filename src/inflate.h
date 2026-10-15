#ifndef INFLATE_H
#define INFLATE_H

// DEFLATE, decompressed: RFC 1951 stored, fixed and dynamic blocks, plus the
// RFC 1950 two-byte zlib wrapper. Taken from OpenNote (src/core/inflate.c at
// 9c2f1f6, same author, MIT). Here it reads the deflate entries of .zip
// archives; a zlib dependency for one algorithm was not worth it.

#include <windows.h>

#ifdef __cplusplus
extern "C" {
#endif

// Decompress a zlib stream (the two-byte header, then DEFLATE). Returns the
// bytes, which the caller frees, or NULL if the data is not a stream this
// reads. `outLen` is what came out.
BYTE* Inflate_Zlib(const BYTE* data, size_t len, size_t* outLen);

// The same for raw DEFLATE, with no wrapper around it.
BYTE* Inflate_Raw(const BYTE* data, size_t len, size_t* outLen);

// Self-check, run by `optics-check --selftest`.
BOOL Inflate_SelfTest(char* failure, size_t failureSize);

#ifdef __cplusplus
}
#endif

#endif // INFLATE_H
