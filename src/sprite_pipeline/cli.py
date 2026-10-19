from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from typing import Sequence

from PIL import Image

from sprite_pipeline.vox import write_vox_manifest

VIEW_ORDER = ("left", "front", "right", "back")


def prepare_four_view(
    views: dict[str, Path], output: Path
) -> Path:
    """Pack four same-sized images horizontally and write a provenance manifest."""
    images: dict[str, Image.Image] = {}
    sizes: set[tuple[int, int]] = set()
    for name in VIEW_ORDER:
        path = views[name]
        with Image.open(path) as source:
            image = source.convert("RGBA")
        images[name] = image
        sizes.add(image.size)

    if len(sizes) != 1:
        dimensions = ", ".join(f"{name}={images[name].width}x{images[name].height}" for name in VIEW_ORDER)
        raise ValueError(f"all four views must have identical dimensions ({dimensions})")

    width, height = images["left"].size
    sheet = Image.new("RGBA", (width * len(VIEW_ORDER), height), (0, 0, 0, 0))
    manifest_views = []
    for index, name in enumerate(VIEW_ORDER):
        sheet.paste(images[name], (index * width, 0))
        source_path = views[name]
        manifest_views.append(
            {
                "direction": name,
                "source": source_path.name,
                "sha256": hashlib.sha256(source_path.read_bytes()).hexdigest(),
                "offset_x": index * width,
            }
        )

    output.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(output)
    manifest_path = output.with_suffix(".manifest.json")
    manifest_path.write_text(
        json.dumps(
            {
                "format": "bn3d-four-view-sheet-v1",
                "view_order": list(VIEW_ORDER),
                "dimensions_per_view": {"width": width, "height": height},
                "output": output.name,
                "views": manifest_views,
                "voxelizer": {
                    "project": "https://github.com/GazPrash/2d-to-3d-voxelizer",
                    "mode": "quad",
                },
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    return manifest_path


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="sprite-pipeline")
    commands = parser.add_subparsers(dest="command", required=True)
    four_view = commands.add_parser(
        "prepare-four-view", help="pack equal-sized directional PNGs for quad mode"
    )
    for name in VIEW_ORDER:
        four_view.add_argument(f"--{name}", type=Path, required=True)
    four_view.add_argument("--output", type=Path, required=True)

    inspect = commands.add_parser(
        "inspect-vox", help="validate a single-model .vox asset and write its manifest"
    )
    inspect.add_argument("--input", type=Path, required=True)
    inspect.add_argument("--manifest", type=Path)
    inspect.add_argument(
        "--asset-kind", choices=("vehicle-part", "character"), required=True
    )
    inspect.add_argument("--asset-id")

    voxelize = commands.add_parser(
        "voxelize",
        help="convert four sprite views or a prepared quad sheet to a renderer-ready VOX model",
    )
    voxelize.add_argument("--sheet", type=Path, help="prepared [LEFT, FRONT, RIGHT, BACK] quad sheet")
    for name in VIEW_ORDER:
        voxelize.add_argument(f"--{name}", type=Path, help=f"{name} view PNG")
    voxelize.add_argument("--output", type=Path, required=True, help="output MagicaVoxel .vox file")
    voxelize.add_argument("--voxel-scale", type=float, default=1.0)
    voxelize.add_argument(
        "--asset-kind", choices=("vehicle-part", "character"), required=True
    )
    voxelize.add_argument("--asset-id")
    voxelize.add_argument("--force", action="store_true")

    kenney = commands.add_parser(
        "kenney-export",
        help="headlessly load a Kenney Shape project and export a renderer-ready VOX model",
    )
    kenney.add_argument(
        "--game-dir", type=Path, required=True,
        help="Kenney Shape installation with MelonLoader installed",
    )
    kenney.add_argument("--project", type=Path, required=True, help="input .kenshape project")
    kenney.add_argument("--output", type=Path, required=True, help="output MagicaVoxel .vox file")
    kenney.add_argument(
        "--asset-kind", choices=("vehicle-part", "character"), required=True
    )
    kenney.add_argument("--asset-id")
    kenney.add_argument(
        "--timeout", type=int, default=180,
        help="maximum game runtime in seconds (default: 180)",
    )
    kenney.add_argument(
        "--force", action="store_true",
        help="replace an existing output file and manifest",
    )
    return parser


def run_voxelizer_backend(input_path: Path, output_path: Path, voxel_scale: float) -> None:
    repo_root = Path(__file__).resolve().parents[2]
    backend_dir = repo_root / "tools" / "voxelizer-backend"
    command = [
        "go",
        "run",
        "./cmd/voxelizer",
        "--input",
        str(input_path),
        "--output",
        str(output_path),
        "--voxel-scale",
        str(voxel_scale),
    ]
    environment = os.environ.copy()
    if not environment.get("GOCACHE"):
        go_cache = Path(tempfile.gettempdir()) / "bn3d-asset-pipeline-go-cache"
        go_cache.mkdir(parents=True, exist_ok=True)
        environment["GOCACHE"] = str(go_cache)
    try:
        process = subprocess.run(
            command,
            cwd=backend_dir,
            env=environment,
            check=False,
            capture_output=True,
            text=True,
        )
    except OSError as error:
        raise RuntimeError("Go is required for the sprite-pipeline voxelize stage") from error
    if process.returncode != 0:
        raise RuntimeError(
            f"voxelizer backend failed:\n{process.stdout[-2000:]}\n{process.stderr[-2000:]}"
        )
    if process.stdout.strip():
        print(process.stdout.strip())
    if process.stderr.strip():
        print(process.stderr.strip(), file=sys.stderr)


def voxelize(args: argparse.Namespace) -> Path:
    output = args.output.expanduser().resolve()
    manifest = output.with_suffix(".manifest.json")
    if output.suffix.lower() != ".vox":
        raise ValueError("--output must use the .vox extension")
    if not args.force and (output.exists() or manifest.exists()):
        raise ValueError("output or manifest already exists; pass --force to replace it")
    if not 1 <= args.voxel_scale <= 1024:
        raise ValueError("--voxel-scale must be between 1 and 1024")

    views = {name: getattr(args, name) for name in VIEW_ORDER}
    provided_views = [name for name, path in views.items() if path is not None]
    if args.sheet is not None and provided_views:
        raise ValueError("use either --sheet or all four directional views")
    if args.sheet is None and len(provided_views) not in (0, len(VIEW_ORDER)):
        missing = ", ".join(name for name in VIEW_ORDER if views[name] is None)
        raise ValueError(f"all four directional views are required; missing: {missing}")
    if args.sheet is None and not provided_views:
        raise ValueError("provide --sheet or all four directional views")

    temporary_sheet = None
    if args.sheet is not None:
        input_path = args.sheet.expanduser().resolve()
        if not input_path.is_file():
            raise ValueError("--sheet must name an existing image")
    else:
        resolved_views = {name: path.expanduser().resolve() for name, path in views.items()}
        with tempfile.TemporaryDirectory(prefix="bn3d-sprite-pipeline-") as temporary_directory:
            temporary_sheet = Path(temporary_directory) / "views.png"
            prepare_four_view(resolved_views, temporary_sheet)
            output.parent.mkdir(parents=True, exist_ok=True)
            run_voxelizer_backend(temporary_sheet, output, args.voxel_scale)
    if temporary_sheet is None:
        output.parent.mkdir(parents=True, exist_ok=True)
        run_voxelizer_backend(input_path, output, args.voxel_scale)

    write_vox_manifest(output, manifest, asset_kind=args.asset_kind, asset_id=args.asset_id)
    return manifest


def kenney_export(args: argparse.Namespace) -> Path:
    if sys.platform != "linux":
        raise ValueError("headless Kenney export currently supports the tested Linux Mono build only")
    if args.timeout < 1:
        raise ValueError("--timeout must be greater than zero")

    game_dir = args.game_dir.expanduser().resolve()
    project = args.project.expanduser().resolve()
    output = args.output.expanduser().resolve()
    if not project.is_file() or project.suffix.lower() != ".kenshape":
        raise ValueError("--project must name an existing .kenshape project")
    if not game_dir.is_dir():
        raise ValueError("--game-dir must name an installed Kenney Shape directory")
    executable = game_dir / "Kenney Shape.x86_64"
    melon_dir = game_dir / "MelonLoader" / "net35"
    mods_dir = game_dir / "Mods"
    bootstrap = game_dir / "MelonLoader.Bootstrap.so"
    managed_game = game_dir / "Kenney Shape_Data" / "Managed" / "Assembly-CSharp.dll"
    if not executable.is_file() or not managed_game.is_file():
        raise ValueError("--game-dir must contain the Linux Kenney Shape Mono installation")
    if not bootstrap.is_file() or not (melon_dir / "MelonLoader.dll").is_file():
        raise ValueError("install and launch MelonLoader once in Kenney Shape before using kenney-export")
    if output.suffix.lower() != ".vox":
        raise ValueError("--output must use the .vox extension")
    manifest = output.with_suffix(".manifest.json")
    if not args.force and (output.exists() or manifest.exists()):
        raise ValueError("output or manifest already exists; pass --force to replace it")

    repo_root = Path(__file__).resolve().parents[2]
    project_file = repo_root / "adapters" / "kenney-shape-probe" / "KenneyShapeProbe.csproj"
    mod_source = repo_root / "adapters" / "kenney-shape-probe" / "bin" / "Release" / "netstandard2.1" / "KenneyShapeProbe.dll"
    mod_target = mods_dir / "KenneyShapeProbe.dll"
    log_path = output.with_suffix(".kenney.log")
    mods_dir.mkdir(parents=True, exist_ok=True)
    try:
        build = subprocess.run(
            [
                "dotnet",
                "build",
                str(project_file),
                "--configuration",
                "Release",
                f"-p:MELONLOADER_DIR={melon_dir}",
                "--verbosity",
                "quiet",
            ],
            cwd=repo_root,
            check=False,
            capture_output=True,
            text=True,
        )
    except OSError as error:
        raise RuntimeError("dotnet is required to build the local Kenney adapter") from error
    if build.returncode != 0 or not mod_source.is_file():
        raise RuntimeError(
            f"could not build Kenney adapter with dotnet:\n{build.stdout}\n{build.stderr}"
        )
    shutil.copy2(mod_source, mod_target)

    output.parent.mkdir(parents=True, exist_ok=True)
    environment = os.environ.copy()
    environment["LD_LIBRARY_PATH"] = str(game_dir) + os.pathsep + environment.get(
        "LD_LIBRARY_PATH", ""
    )
    environment["LD_PRELOAD"] = str(bootstrap) + (
        " " + environment["LD_PRELOAD"] if environment.get("LD_PRELOAD") else ""
    )
    environment["BN3D_KENNEY_PROJECT_PATH"] = str(project)
    environment["BN3D_KENNEY_EXPORT_PATH"] = str(output)
    try:
        process = subprocess.run(
            [str(executable), "-batchmode", "-nographics", "-logFile", str(log_path)],
            cwd=game_dir,
            env=environment,
            check=False,
            capture_output=True,
            text=True,
            timeout=args.timeout,
        )
    except subprocess.TimeoutExpired as error:
        raise RuntimeError(f"Kenney Shape exceeded {args.timeout} seconds; inspect {log_path}") from error
    if process.returncode != 0:
        raise RuntimeError(
            f"Kenney Shape exited with status {process.returncode}; inspect {log_path}\n"
            f"{process.stderr[-2000:]}"
        )
    if not output.is_file() or output.stat().st_size == 0:
        raise RuntimeError(f"Kenney Shape did not create the requested output; inspect {log_path}")

    write_vox_manifest(output, manifest, asset_kind=args.asset_kind, asset_id=args.asset_id)
    return manifest


def main(argv: Sequence[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        if args.command == "prepare-four-view":
            manifest = prepare_four_view(
                {name: getattr(args, name) for name in VIEW_ORDER}, args.output
            )
            print(f"Wrote {args.output} and {manifest}")
        elif args.command == "inspect-vox":
            manifest_path = args.manifest or args.input.with_suffix(".manifest.json")
            write_vox_manifest(
                args.input,
                manifest_path,
                asset_kind=args.asset_kind,
                asset_id=args.asset_id,
            )
            print(f"Validated {args.input}; wrote {manifest_path}")
        elif args.command == "voxelize":
            manifest = voxelize(args)
            print(f"Validated {args.output}; wrote {manifest}")
        else:
            manifest = kenney_export(args)
            print(f"Headlessly exported and validated {args.output}; wrote {manifest}")
    except (OSError, ValueError, RuntimeError) as error:
        print(f"sprite-pipeline: error: {error}")
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
