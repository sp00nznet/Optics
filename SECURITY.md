# Security

Optics reads files it did not write: models, archives and images from anywhere. That
is the attack surface, and the only one: it opens no network connections, runs nothing
from the files it reads, and writes only the last-used folder path to
`HKCU\Software\Optics\LastPath`.

What guards the parsers:
- The zip reader bounds-checks the central directory against the file size, skips
  encrypted and zip64 members, and rejects a member whose inflated size differs from the
  one recorded.
- DEFLATE (src/inflate.c) checks every read against the input and grows output with
  bounds.
- FBX and OBJ go through ufbx, which is fuzzed upstream; images through stb_image.

## Reporting
Please report vulnerabilities privately through GitHub's
[security advisories](https://github.com/sp00nznet/Optics/security/advisories/new) rather
than a public issue. A file that crashes Optics is welcome as a report.

## Binaries
Release binaries are built by GitHub Actions from the tagged commit and are not
code-signed, so SmartScreen warns on first run.
