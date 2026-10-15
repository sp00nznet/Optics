# Architecture

Optics is one C program in two executables over one static library.

```
 Optics.exe (src/main.c)          optics-check.exe (src/check.c)
   window, list, 3D view            --selftest, --check, --render
            \                          /
             +------ core (static) ---+
             | scan.c    walk a folder, list zip members        |
             | zip.c     read the central directory, one member |
             | inflate.c DEFLATE (from OpenNote)                |
             | model.c   STL here; FBX/OBJ via ufbx; textures   |
             | render.c  OpenGL 1.1 draw; DIB render to PNG     |
             +-------------------------------------------------+
                         third_party: ufbx, stb_image(_write)
```

## Data flow

1. **Scan** (`Scan`, on a worker thread in the GUI) walks the chosen folder with
   `FindFirstFileExW`, records every `.stl .obj .fbx .scad .3mf .step .stp`, and opens each
   `.zip` to list its members without extracting them. The result is a flat array of `Item`
   (path, member name, relative path, size, kind). Nothing is written anywhere: the library
   *is* the folder, so a rescan is the only refresh there is.
2. **Select** an item and `Model_Load` reads its bytes (from disk, or one member out of the
   zip), parses it, and flattens it into `Part`s: non-indexed triangles in the
   `GL_T2F_N3F_V3F` layout, one part per (mesh instance, material). Textures are decoded to
   RGBA once per texture and shared between parts.
3. **Draw**: `Render_Draw` uploads textures on first use and draws each part with
   `glInterleavedArrays` + `glDrawArrays`.
4. **Thumbnails**: when the list asks for a row's image, the item is copied onto a queue.
   One worker thread takes the newest request first, loads the model and draws it with
   `Render_Offscreen` (the software GL `--render` uses, safe off the UI thread), and posts
   the bitmap back. Newest-first means the rows on screen always render before the ones
   scrolled past. A rescan bumps a generation number so late results for the old library
   are dropped.

## Why these choices

- **No database, no server, no users.** The web version indexed into PostgreSQL behind a
  login. For one person looking at their own folder that was all overhead; a walk of tens
  of thousands of files takes seconds on a warm cache.
- **OpenGL 1.1 fixed function.** It needs no loader, works on every Windows install
  including Microsoft's software renderer over RDP and in VMs, and is the only GL a DIB
  section offers, which is what makes `--render` work with no window or display. The cost:
  textures are resampled to power-of-two sizes and there are no mipmaps.
- **ufbx** for FBX and OBJ: one C file, MIT, reads binary and ASCII FBX 6/7 and OBJ+MTL,
  and hands back each material's texture reference plus embedded texture bytes.
  Finding texture files is ours: see [textures.md](textures.md).
- **Two executables.** A GUI-subsystem program has no stdout and the shell does not wait
  for it, so the CI harness is a separate console program over the same library.
- **Static CRT.** The exe runs on a machine with no VC++ redistributable.

## Not here

Duplicate detection and tags from the web version are not carried over; see
[ROADMAP.md](../ROADMAP.md).
