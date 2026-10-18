from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
from typing import Sequence

from PIL import Image

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
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        manifest = prepare_four_view(
            {name: getattr(args, name) for name in VIEW_ORDER}, args.output
        )
    except (OSError, ValueError) as error:
        print(f"sprite-pipeline: error: {error}")
        return 2
    print(f"Wrote {args.output} and {manifest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
