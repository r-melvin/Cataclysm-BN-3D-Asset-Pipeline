from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from sprite_pipeline.cli import prepare_four_view


class FourViewPreparationTest(unittest.TestCase):
    def test_packs_views_in_voxelizer_order_and_records_hashes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            paths = {}
            colors = {
                "left": (255, 0, 0, 255),
                "front": (0, 255, 0, 255),
                "right": (0, 0, 255, 255),
                "back": (255, 255, 0, 255),
            }
            for name, color in colors.items():
                path = root / f"{name}.png"
                Image.new("RGBA", (2, 3), color).save(path)
                paths[name] = path

            output = root / "out" / "character.png"
            manifest_path = prepare_four_view(paths, output)
            with Image.open(output) as sheet:
                self.assertEqual(sheet.size, (8, 3))
                self.assertEqual([sheet.getpixel((x * 2, 0)) for x in range(4)], list(colors.values()))
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
            self.assertEqual(manifest["view_order"], ["left", "front", "right", "back"])
            self.assertEqual([view["offset_x"] for view in manifest["views"]], [0, 2, 4, 6])
            self.assertTrue(all(len(view["sha256"]) == 64 for view in manifest["views"]))

    def test_rejects_different_source_dimensions(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            paths = {}
            for index, name in enumerate(("left", "front", "right", "back")):
                path = root / f"{name}.png"
                Image.new("RGBA", (2 + int(index == 3), 3), (0, 0, 0, 0)).save(path)
                paths[name] = path
            with self.assertRaisesRegex(ValueError, "identical dimensions"):
                prepare_four_view(paths, root / "sheet.png")


if __name__ == "__main__":
    unittest.main()
