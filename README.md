# Bright Nights 3D asset pipeline

A separate, offline-friendly project for preparing directional sprite art and authored voxel assets for the Cataclysm: Bright Nights 3D renderer.

## Current workflow

- `sprite-pipeline prepare-four-view` validates and packs left, front, right, and back PNGs into the horizontal sheet expected by [2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer). That application currently exports OBJ; importing its output into the BN renderer is a separate roadmap item.
- `sprite-pipeline inspect-vox` validates a single-model MagicaVoxel file against the subset currently consumed by the BN renderer and writes a SHA-256 asset manifest.
- [Kenney Shape](https://kenney.nl/tools/kenney-shape) is an optional authoring tool. It can export MagicaVoxel `.vox` files that the renderer can load for vehicle parts. Install it separately; this repository does not include or redistribute the Kenney Shape ZIP, executable, or other application files.
- An optional MelonLoader adapter lives in `adapters/kenney-shape-probe/`. For the local Unity 2021.3.45f2 Mono build, F8 invokes Kenney Shape's `ExportVoxel.Export()` path and redirects its save dialog to a timestamped `.vox` file. The adapter is isolated to Kenney Shape and is never loaded by BN. The adapter compiles against the app's Mono `net35` MelonLoader assemblies; runtime loading and an actual in-app export still need visual confirmation. See its README for setup and limits.

These tools are not yet one end-to-end converter: the voxelizer exports OBJ and Kenney Shape exports `.vox`, and Kenney Shape does not document OBJ import. The pipeline keeps those paths explicit instead of silently changing model data.

## Install and run

Requires Python 3.11+ and Pillow.

```sh
python -m pip install -e .
sprite-pipeline prepare-four-view \
  --left path/to/left.png \
  --front path/to/front.png \
  --right path/to/right.png \
  --back path/to/back.png \
  --output build/character-quad.png
```

Inputs must have identical dimensions. The output is RGBA PNG in `[LEFT, FRONT, RIGHT, BACK]` order, with an adjacent JSON manifest recording dimensions, input names, hashes, and the required tool settings. No images are resized or recolored.

Validate a `.vox` model exported from Kenney Shape before handing it to the renderer:

```sh
sprite-pipeline inspect-vox \
  --input path/to/wheel.vox \
  --asset-kind vehicle-part \
  --asset-id wheel
```

The command accepts either `vehicle-part` or `character`, rejects malformed voxel coordinates and multi-model scene files the renderer does not yet support, and writes `wheel.manifest.json` beside the model by default. The manifest records the source hash, dimensions, voxel count, and compatibility subset.

Run checks with:

```sh
python -m unittest discover -s tests -v
```

## Asset handoff to BN

For a vehicle part, export an authored `.vox` from Kenney Shape and copy it to BN's `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. The renderer uses that model for the matching vehicle part; missing assets continue to use its normal renderer path.

Optional voxel item models belong at `data/raw/r3d/items/<item-type-id>.vox`. Enable the renderer's **R3D voxel items** option to display them; without an asset, items use the normal renderer path. Keep source projects and licenses with the asset during development. Do not put editor applications or their archives in the BN game data directory.

Character models and OBJ import are future work. The four-view image packer currently prepares input for the external voxelizer, but its OBJ output is not accepted by the renderer. Do not treat a generated model as renderer-ready until a supported import path exists and it has been checked in-game from all four directions.

## Project structure

- `src/sprite_pipeline/`: command-line preparation and validation.
- `tests/`: deterministic unit tests.
- `docs/roadmap.md`: integration milestones and acceptance checks.
- `docs/tools.md`: tool boundaries, installation links, and licensing notes.
