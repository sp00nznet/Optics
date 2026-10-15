# Conformance harness

```
optics-check --check sample-models tests/expected.tsv
```

Scans the folder exactly as the app does, loads every STL, OBJ and FBX in it (including
zip members), and compares each with `tests/expected.tsv`: triangle count, and textures
loaded out of textures wanted. A line is

```
PASS Textured/store/source/crate.fbx  tris=74 tex=1/1
```

It **fails** (exit 1) when a file that loaded no longer does, when a count changes, or when a
file in the expected list is gone. A new file that loads is a PASS marked `(new)` until the
expected list is refreshed with `--update`. CI runs it on every push.

Current: **11/11** on the committed corpus (6 third-party STLs, 5 generated textured
FBX/OBJ covering each texture-finding route in [textures.md](textures.md)).

## Larger corpora

Point it at any folder with an expected file of your own:

```
optics-check --check D:\models D:\models-expected.tsv --update   # record today's numbers
optics-check --check D:\models D:\models-expected.tsv            # later: what regressed
```

Libraries of purchased or downloaded models can't be committed (repo rules, section 3),
so they stay outside the repo.
