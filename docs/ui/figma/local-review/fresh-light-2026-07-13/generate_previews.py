#!/usr/bin/env python3
"""Generate local PNG/PDF artifacts from the committed Fresh Light HTML/SVG preview.

Run from the repository root:
    python docs/ui/figma/local-review/fresh-light-2026-07-13/generate_previews.py

Requires Chromium/Chrome. The script never reads application data and only renders the
synthetic preview committed beside this file.
"""
from __future__ import annotations

import hashlib
import json
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent
HTML = ROOT / "Fresh-Light预览.html"
SVG = ROOT / "fresh-light-all-frames.svg"
PNG = ROOT / "fresh-light-all-frames.png"
PDF = ROOT / "fresh-light-preview.pdf"
CHECKSUMS = ROOT / "sha256sums.txt"


def browser() -> str:
    for name in ("chromium", "chromium-browser", "google-chrome", "google-chrome-stable"):
        path = shutil.which(name)
        if path:
            return path
    raise SystemExit("Chromium/Chrome not found. Install one or open the HTML manually.")


def run() -> None:
    exe = browser()
    url = HTML.as_uri()
    common = [exe, "--headless=new", "--no-sandbox", "--disable-gpu", "--hide-scrollbars"]
    subprocess.run(common + ["--window-size=1480,2040", f"--screenshot={PNG}", url], check=True)
    subprocess.run(common + [f"--print-to-pdf={PDF}", "--no-pdf-header-footer", url], check=True)

    files = [HTML, SVG, PNG, PDF, ROOT / "fresh-light-preview-manifest.json", Path(__file__)]
    lines = []
    for path in files:
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        lines.append(f"{digest}  {path.name}")
    CHECKSUMS.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(json.dumps({"png": str(PNG), "pdf": str(PDF), "checksums": str(CHECKSUMS)}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    run()
