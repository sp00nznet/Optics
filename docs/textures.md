# Finding an FBX's textures

The format part of FBX is solved by ufbx. What took the work is that an FBX records the
texture path *of the machine that exported it*, which almost never exists on this one.
`Related()` in [src/model.c](../src/model.c) tries, in order, and stops at the first hit:

1. **Embedded bytes.** FBX exported with "embed textures" carries the image inside it
   (`ufbx_texture.content`). Nothing to find.
2. **The stated path.** The relative path, the resolved filename, then the absolute path.
   An absolute path works only on the artist's own machine, but costs nothing to try.
3. **Beside the model**, by file name alone. Most downloads flatten the folder.
4. **`textures/` beside it, and `../textures/`.** The second is the layout Sketchfab and most
   asset stores ship: `source/model.fbx` next to `textures/*.png`.
5. **Each of 3 and 4 again with a different extension** (`.png .jpg .jpeg .tga .bmp .psd`).
   Textures get converted far more often than renamed: an FBX that says `wood.tga` next to
   a `wood.png` is common.

Inside a zip, steps 2 to 5 become: the stated path relative to the model's folder in the
archive, then any member with the same file name (any folder), then the same with other
extensions.

An OBJ's `.mtl` is found the same way, because ufbx asks for it through the same callback
(`OpenRelated`).

## What a miss looks like

The model still loads and draws in its material colour. The status bar says
`textures 0 of 1, missing no_such_texture.png`, and `optics-check --render` prints the same.
`sample-models/Textured/missing/crate.fbx` is that case, in the conformance corpus.

## Known gaps

- **DDS** textures are not decoded (stb_image has no DDS). Game rips use them a lot.
- Only the base colour / diffuse map is used. Normal, roughness and other maps are ignored.
- UV `v` is flipped once at decode (`stbi_set_flip_vertically_on_load`): FBX and OBJ put
  `v = 0` at the bottom of the image. The crate texture's quadrants (red top-left, green
  top-right, blue bottom-left, yellow bottom-right) make a flip or mirror obvious.
