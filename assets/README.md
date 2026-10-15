# Icon

A cube in sunglasses holding a magnifying glass, because
[futureburn](https://github.com/sp00nznet/futureburn) is a CD in sunglasses and
[vncfree](https://github.com/sp00nznet/vncfree) is a monitor in sunglasses, and these are
siblings. A cube for 3D models, the glass for looking through a library of them.

| File | What it is |
|---|---|
| `optics.png` | The master, 512px with a transparent background. |
| `optics.ico` | 16, 24, 32, 48, 64, 128 and 256px, built from the master. |
| `optics.rc` | One line naming the icon as resource `1`, linked into `Optics.exe` by CMake. |

Resource `1` is deliberate. Explorer and the taskbar use the lowest-numbered icon in a
binary, so a higher number is quietly ignored.

## Redrawing it

The master came out of [asset-forge](https://github.com/sp00nznet/asset-forge)'s `txt2img`
on a local ComfyUI, preset `zimage_turbo` (Z-Image Turbo, Apache-2.0) at 1024x1024,
**seed 70423**, with the prompt in `icon_prompt.jsonl`. From the asset-forge checkout:

    python -m pipelines.txt2img --prompts <optics>/assets/icon_prompt.jsonl --preset zimage_turbo --width 1024 --height 1024 --seed 70423 --out-dir out
    py <optics>/tools/make_icon.py out/optics_icon_0.png

Two things that matter if you regenerate, both learned on vncfree:

- **Cut the background out by flooding in from the edges, not by keying out white.** The
  sneakers, gloves and lens highlights are white too, and a colour key punches holes
  straight through them. The black outline is what stops the fill getting inside.
- **Crop to the artwork before downscaling.** A render leaves a wide margin, and keeping
  it means the mascot is a handful of pixels across by the time it reaches 16px.
