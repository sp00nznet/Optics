"""Generate the textured sample models in sample-models/Textured and Archives.

Run with Blender (tested with 5.0):

    blender -b --factory-startup --python tools/make_samples.py

Everything is made here from code: a two-material "crate" whose albedo is a
quadrant pattern (red top-left, green top-right, blue bottom-left, yellow
bottom-right), so a flipped or mirrored UV shows up at a glance. Each export
exercises one way of finding a texture (docs/textures.md):

  Textured/crate.fbx            texture beside the model, relative path
  Textured/embedded/crate.fbx   texture embedded in the FBX, no copy beside it
  Textured/crate.obj (+ .mtl)   OBJ/MTL, texture beside it
  Textured/store/source/crate.fbx
                                names crate_albedo.tga with no folder; only
                                ../textures/crate_albedo.png exists, the
                                asset-store layout plus a converted extension
  Textured/missing/crate.fbx    names a texture that exists nowhere: must
                                still load, and report 0 of 1 textures
  Archives/textured_crate.zip   FBX and texture inside a zip

Export happens inside the repo on purpose: Blender writes the export path into
the FBX header, and a path under a user profile would leak the user name.
"""
import os
import shutil
import sys
import zipfile

import bpy

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "sample-models"))
TEX = os.path.join(ROOT, "Textured")
SIZE = 256


def albedo(path):
    img = bpy.data.images.new("crate_albedo", SIZE, SIZE)
    quad = {(0, 1): (0.85, 0.2, 0.2), (1, 1): (0.2, 0.75, 0.25),
            (0, 0): (0.2, 0.35, 0.9), (1, 0): (0.95, 0.8, 0.15)}
    px = []
    for y in range(SIZE):  # Blender rows run bottom-up
        for x in range(SIZE):
            r, g, b = quad[(x * 2 // SIZE, y * 2 // SIZE)]
            k = 1.0 if ((x // 32) + (y // 32)) % 2 else 0.75
            edge = 0.15 if x % 128 < 4 or y % 128 < 4 else 1.0
            px += [r * k * edge, g * k * edge, b * k * edge, 1.0]
    img.pixels = px
    img.filepath_raw = path
    img.file_format = "PNG"
    img.save()
    return img


def material(name, img=None, color=None):
    mat = bpy.data.materials.new(name)
    if not mat.node_tree:  # Blender < 5 needs use_nodes switched on
        mat.use_nodes = True
    bsdf = next(n for n in mat.node_tree.nodes if n.type == "BSDF_PRINCIPLED")
    if img:
        node = mat.node_tree.nodes.new("ShaderNodeTexImage")
        node.image = img
        mat.node_tree.links.new(node.outputs["Color"], bsdf.inputs["Base Color"])
    else:
        bsdf.inputs["Base Color"].default_value = (*color, 1.0)
    return mat


def build(img):
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.object.delete()
    bpy.ops.mesh.primitive_cube_add(size=2, location=(0, 0, 1))
    crate = bpy.context.object
    crate.name = "crate"
    crate.data.materials.append(material("crate_wood", img=img))
    # Each face gets the whole texture, so every side shows all four quadrants.
    uv = crate.data.uv_layers.active.data
    for poly in crate.data.polygons:
        for li, (u, v) in zip(poly.loop_indices, [(0, 0), (1, 0), (1, 1), (0, 1)]):
            uv[li].uv = (u, v)
    bpy.ops.mesh.primitive_cone_add(radius1=0.6, depth=1.2, location=(0, 0, 2.6))
    cone = bpy.context.object
    cone.name = "marker"
    cone.data.materials.append(material("marker_orange", color=(1.0, 0.45, 0.05)))
    bpy.ops.object.select_all(action="SELECT")


def fbx(path, **kw):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    bpy.ops.export_scene.fbx(filepath=path, use_selection=True, **kw)


def main():
    os.makedirs(TEX, exist_ok=True)
    png = os.path.join(TEX, "crate_albedo.png")
    img = albedo(png)
    build(img)

    fbx(os.path.join(TEX, "crate.fbx"), path_mode="RELATIVE")
    emb = os.path.join(TEX, "embedded")
    fbx(os.path.join(emb, "crate.fbx"), path_mode="COPY", embed_textures=True)
    for leftover in [f for f in os.listdir(emb) if not f.endswith(".fbx")]:
        os.remove(os.path.join(emb, leftover))  # only the embedded copy may count
    bpy.ops.wm.obj_export(filepath=os.path.join(TEX, "crate.obj"), export_selected_objects=True,
                          export_materials=True, path_mode="RELATIVE")

    store = os.path.join(TEX, "store")
    img.filepath_raw = "//crate_albedo.tga"
    fbx(os.path.join(store, "source", "crate.fbx"), path_mode="STRIP")
    img.filepath_raw = "//no_such_texture.png"
    fbx(os.path.join(TEX, "missing", "crate.fbx"), path_mode="STRIP")
    img.filepath_raw = png
    os.makedirs(os.path.join(store, "textures"), exist_ok=True)
    shutil.copy(png, os.path.join(store, "textures", "crate_albedo.png"))

    # A copy with the embedded texture would make the zip test meaningless.
    with zipfile.ZipFile(os.path.join(ROOT, "Archives", "textured_crate.zip"), "w", zipfile.ZIP_DEFLATED) as z:
        z.write(os.path.join(TEX, "crate.fbx"), "crate/crate.fbx")
        z.write(png, "crate/crate_albedo.png")

    print("samples written to", ROOT)


if __name__ == "__main__":
    main()
    sys.exit(0)
