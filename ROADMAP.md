# Roadmap

## Next
- **Thumbnail cache**: thumbnails are rendered again each run; cache them under
  `%LOCALAPPDATA%` if a big library makes that slow.
- **DDS textures**: common in game assets; stb_image does not read them.
- **3MF preview**: a zip of XML meshes; the zip reader is already here.
- **Load off the UI thread**: a large FBX freezes the window while it loads.

## Later
- Duplicate detection by content hash (the web version had it).
- Normal maps and other material channels, which needs GL 2.0+.
- An installer with "Open with Optics" for the model extensions.
- zip64 archives (members of zips over 4 GB are skipped today).
- Animation playback for rigged FBX.

## Out of scope
- User accounts, a server, or a database. The library is the folder.
- Editing or converting models.
- STEP preview: needs a B-rep kernel (OpenCascade is LGPL, see repo rules on copyleft).
