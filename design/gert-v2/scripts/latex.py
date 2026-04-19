#!/usr/bin/env python3
"""Cross-platform LaTeX build helper for the Gert v2 design docs."""

from __future__ import annotations

import argparse
import shutil
import subprocess
import sys
import zipfile
from datetime import datetime
from pathlib import Path


def find_executable(name: str) -> str | None:
    return shutil.which(name)


def run(cmd: list[str], cwd: Path) -> int:
    print("Running:", " ".join(cmd))
    proc = subprocess.run(cmd, cwd=str(cwd))
    return proc.returncode


def select_engine(requested: str) -> str | None:
    if requested != "auto":
        return requested if find_executable(requested) else None

    for candidate in ("tectonic", "latexmk", "pdflatex"):
        if find_executable(candidate):
            return candidate
    return None


def cmd_check(args: argparse.Namespace) -> int:
    project_dir = Path(__file__).resolve().parent.parent
    main_file = project_dir / args.main

    if not main_file.exists():
        print(f"error: main tex file not found: {main_file}")
        return 1

    engine = select_engine(args.engine)
    if engine is None:
        print("error: no supported LaTeX engine found in PATH")
        print("supported engines: tectonic, latexmk, pdflatex")
        print("install tectonic: https://tectonic-typesetting.github.io/")
        return 1

    print(f"ok: using engine: {engine}")
    print(f"ok: main tex file exists: {main_file}")
    return 0


def build_with_engine(engine: str, args: argparse.Namespace, project_dir: Path, build_dir: Path) -> int:
    if engine == "tectonic":
        cmd = ["tectonic", "--keep-logs", "--outdir", str(build_dir), args.main]
        return run(cmd, cwd=project_dir)

    if engine == "latexmk":
        cmd = [
            "latexmk",
            "-pdf",
            "-interaction=nonstopmode",
            "-shell-escape",
            f"-outdir={build_dir}",
            args.main,
        ]
        return run(cmd, cwd=project_dir)

    if engine == "pdflatex":
        # Two passes for references/table of contents.
        cmd = [
            "pdflatex",
            "-interaction=nonstopmode",
            "--shell-escape",
            f"-output-directory={build_dir}",
            args.main,
        ]
        code = run(cmd, cwd=project_dir)
        if code != 0:
            return code
        return run(cmd, cwd=project_dir)

    print(f"error: unsupported engine {engine}")
    return 2


def cmd_build(args: argparse.Namespace) -> int:
    project_dir = Path(__file__).resolve().parent.parent
    build_dir = project_dir / args.build_dir
    build_dir.mkdir(parents=True, exist_ok=True)

    engine = select_engine(args.engine)
    if engine is None:
        return cmd_check(args)

    status = cmd_check(args)
    if status != 0:
        return status

    return build_with_engine(engine, args, project_dir, build_dir)


def cmd_clean(args: argparse.Namespace) -> int:
    project_dir = Path(__file__).resolve().parent.parent
    build_dir = project_dir / args.build_dir

    if not build_dir.exists():
        print(f"nothing to clean: {build_dir}")
        return 0

    for path in build_dir.iterdir():
        if path.is_file():
            path.unlink()
        elif path.is_dir():
            shutil.rmtree(path)

    print(f"cleaned: {build_dir}")
    return 0


def cmd_release(args: argparse.Namespace) -> int:
    project_dir = Path(__file__).resolve().parent.parent
    build_dir = project_dir / args.build_dir
    pdf_file = build_dir / f"{Path(args.main).stem}.pdf"

    if not pdf_file.exists():
        print(f"error: PDF not found: {pdf_file}")
        print("hint: run 'make build' first")
        return 1

    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    zip_name = f"gert-v2-design-{timestamp}.zip"
    zip_path = project_dir / zip_name

    try:
        with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
            # Add PDF
            zf.write(pdf_file, arcname=f"gert-v2-design/{pdf_file.name}")

            # Add source files
            for section_file in sorted((project_dir / "sections").glob("*.tex")):
                zf.write(section_file, arcname=f"gert-v2-design/sections/{section_file.name}")

            zf.write(project_dir / "main.tex", arcname="gert-v2-design/main.tex")
            zf.write(project_dir / "README.md", arcname="gert-v2-design/README.md")

            # Add metadata
            metadata = (
                "# Gert v2 Design Document\n\n"
                f"Built: {datetime.now().isoformat()}\n\n"
                "Contents:\n"
                "- main.pdf: Compiled design document\n"
                "- main.tex: Root LaTeX file\n"
                "- sections/: Individual design sections\n"
                "- README.md: Build and usage instructions\n"
            )
            zf.writestr("gert-v2-design/MANIFEST.md", metadata)

        print(f"ok: created release archive: {zip_path}")
        print(f"    size: {zip_path.stat().st_size / 1024:.1f} KB")
        return 0
    except Exception as e:
        print(f"error: failed to create release archive: {e}")
        return 1


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Build helper for Gert v2 LaTeX docs")
    p.add_argument("command", choices=["check", "build", "clean", "release"])
    p.add_argument("--main", default="main.tex", help="Main LaTeX file")
    p.add_argument("--build-dir", default="build", help="Build output directory")
    p.add_argument(
        "--engine",
        default="auto",
        choices=["auto", "tectonic", "latexmk", "pdflatex"],
        help="LaTeX engine to use",
    )
    return p


def main() -> int:
    args = parser().parse_args()
    if args.command == "check":
        return cmd_check(args)
    if args.command == "build":
        return cmd_build(args)
    if args.command == "clean":
        return cmd_clean(args)
    if args.command == "release":
        return cmd_release(args)

    print("error: unsupported command")
    return 2


if __name__ == "__main__":
    sys.exit(main())
