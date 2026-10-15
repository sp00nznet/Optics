"""Turn an asset-forge render into assets/optics.png and assets/optics.ico.

    py tools/make_icon.py <render.png>

The render comes from asset-forge txt2img (assets/README.md has the command).
Two steps matter, both learned on vncfree's icons:

- The background is removed by flood-filling in from the edges, not by keying
  out white: the sneakers and highlights are white too, and a colour key
  punches holes through them. The black outline stops the fill.
- The artwork is cropped before downscaling, or the mascot is a few pixels
  wide by the time it reaches 16px.

Needs Pillow.
"""
import os
import sys

from PIL import Image, ImageDraw

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "..", "assets")
KEY = (255, 0, 255)


def main(src):
    im = Image.open(src).convert("RGB")
    w, h = im.size
    # Seed the fill all round the border, so a margin broken up by the
    # artwork touching one edge is still reached from the others.
    for x in range(0, w, 16):
        for y in (0, h - 1):
            if im.getpixel((x, y)) != KEY:
                ImageDraw.floodfill(im, (x, y), KEY, thresh=60)
    for y in range(0, h, 16):
        for x in (0, w - 1):
            if im.getpixel((x, y)) != KEY:
                ImageDraw.floodfill(im, (x, y), KEY, thresh=60)

    rgba = im.convert("RGBA")
    rgba.putdata([(0, 0, 0, 0) if p[:3] == KEY else p for p in rgba.getdata()])
    box = rgba.getbbox()
    art = rgba.crop(box)
    side = max(art.size) + 8
    square = Image.new("RGBA", (side, side), (0, 0, 0, 0))
    square.paste(art, ((side - art.width) // 2, (side - art.height) // 2))
    master = square.resize((512, 512), Image.LANCZOS)
    master.save(os.path.join(OUT, "optics.png"), optimize=True)
    master.save(os.path.join(OUT, "optics.ico"),
                sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
    print("wrote assets/optics.png and assets/optics.ico")


if __name__ == "__main__":
    main(sys.argv[1])
