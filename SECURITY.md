# Security Policy

## Supported Versions

| Version | Supported |
|---|---|
| 1.0.x | ✅ |
| < 1.0.0 | ❌ |

## Reporting a Vulnerability

Please open a private security report with:

- affected version
- reproduction steps
- impact assessment

## Known Limitations

- Scanned/image-only PDFs are not OCR-processed.
- Token counts are approximate (`ceil(bytes/4) + word_count`).
- Compression is drop-only and deterministic; no semantic rewriting.
