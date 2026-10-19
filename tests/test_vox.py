from __future__ import annotations

import json
import struct
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from PIL import Image

from sprite_pipeline.cli import build_parser, voxelize
from sprite_pipeline.vox import inspect_vox, write_vox_manifest


def _chunk(chunk_id: bytes, content: bytes, children: bytes = b"") -> bytes:
    return struct.pack("<4sII", chunk_id, len(content), len(children)) + content + children


def _vox_file(
    *,
    dimensions: tuple[int, int, int] = (2, 2, 2),
    voxels: tuple[tuple[int, int, int, int], ...] = ((0, 1, 1, 1),),
    add_second_model: bool = False,
    palette: bytes | None = None,
) -> bytes:
    model_chunks = []
    for _ in range(2 if add_second_model else 1):
        size = _chunk(b"SIZE", struct.pack("<III", *dimensions))
        voxel_content = struct.pack("<I", len(voxels)) + b"".join(
            struct.pack("<BBBB", *voxel) for voxel in voxels
        )
        model_chunks.extend((size, _chunk(b"XYZI", voxel_content)))
    children = b"".join(model_chunks)
    if palette is not None:
        children += _chunk(b"RGBA", palette)
    return b"VOX " + struct.pack("<I", 150) + _chunk(b"MAIN", b"", children)


class VoxInspectionTest(unittest.TestCase):
    def test_inspects_single_model_and_writes_provenance_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            source = root / "wheel.vox"
            source.write_bytes(_vox_file())

            model = inspect_vox(source)
            self.assertEqual(model.version, 150)
            self.assertEqual(model.size, (2, 2, 2))
            self.assertEqual(model.voxel_count, 1)
            self.assertIsNone(model.palette)

            manifest_path = root / "wheel.manifest.json"
            write_vox_manifest(
                source,
                manifest_path,
                asset_kind="vehicle-part",
                asset_id="wheel",
            )
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
            self.assertEqual(manifest["asset_id"], "wheel")
            self.assertEqual(manifest["asset_kind"], "vehicle-part")
            self.assertEqual(manifest["voxel_count"], 1)
            self.assertEqual(len(manifest["sha256"]), 64)

    def test_unified_voxelize_command_packs_calls_backend_and_validates(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            views = {}
            for direction in ("left", "front", "right", "back"):
                path = root / f"{direction}.png"
                Image.new("RGBA", (2, 2), (12, 34, 56, 255)).save(path)
                views[direction] = path
            output = root / "wheel.vox"
            args = build_parser().parse_args(
                [
                    "voxelize",
                    "--left", str(views["left"]),
                    "--front", str(views["front"]),
                    "--right", str(views["right"]),
                    "--back", str(views["back"]),
                    "--output", str(output),
                    "--asset-kind", "vehicle-part",
                    "--asset-id", "wheel",
                ]
            )

            def fake_backend(input_path: Path, output_path: Path, voxel_scale: float) -> None:
                self.assertEqual(input_path.name, "views.png")
                self.assertEqual(voxel_scale, 1.0)
                with Image.open(input_path) as sheet:
                    self.assertEqual(sheet.size, (8, 2))
                output_path.write_bytes(_vox_file())

            with patch("sprite_pipeline.cli.run_voxelizer_backend", side_effect=fake_backend):
                manifest_path = voxelize(args)

            self.assertEqual(manifest_path, output.with_suffix(".manifest.json"))
            self.assertEqual(inspect_vox(output).voxel_count, 1)
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
            self.assertEqual(manifest["asset_id"], "wheel")

    def test_rejects_multiple_models_unsupported_by_renderer(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            source = Path(temp_dir) / "scene.vox"
            source.write_bytes(_vox_file(add_second_model=True))
            with self.assertRaisesRegex(ValueError, "exactly one SIZE"):
                inspect_vox(source)

    def test_rejects_voxels_outside_dimensions_and_duplicate_positions(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            outside = root / "outside.vox"
            outside.write_bytes(_vox_file(voxels=((2, 0, 0, 1),)))
            with self.assertRaisesRegex(ValueError, "outside the SIZE"):
                inspect_vox(outside)

            duplicate = root / "duplicate.vox"
            duplicate.write_bytes(
                _vox_file(voxels=((0, 0, 0, 1), (0, 0, 0, 2)))
            )
            with self.assertRaisesRegex(ValueError, "duplicate voxel"):
                inspect_vox(duplicate)

    def test_rejects_zero_palette_index(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            source = Path(temp_dir) / "transparent.vox"
            source.write_bytes(_vox_file(voxels=((0, 0, 0, 0),)))
            with self.assertRaisesRegex(ValueError, "reserved for transparency"):
                inspect_vox(source)

    def test_accepts_embedded_palette_and_rejects_bad_chunk_lengths(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            source = root / "palette.vox"
            palette = bytes((index % 256 for index in range(1024)))
            source.write_bytes(_vox_file(palette=palette))
            self.assertIsNotNone(inspect_vox(source).palette)

            truncated = root / "truncated.vox"
            truncated.write_bytes(_vox_file()[:-1])
            with self.assertRaisesRegex(ValueError, "MAIN child size"):
                inspect_vox(truncated)


if __name__ == "__main__":
    unittest.main()
