# Bright Nights 3D asset pipeline

A separate, offline-friendly project for preparing directional sprite art and authored voxel assets for the Cataclysm: Bright Nights 3D renderer.

## Current workflow

- `sprite-pipeline prepare-four-view` validates and packs left, front, right, and back PNGs into the horizontal sheet expected by [2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer). That application currently exports OBJ; importing its output into the BN renderer is a separate roadmap item.
- [Kenney Shape](https://kenney.nl/tools/kenney-shape) is an optional authoring tool. It can export MagicaVoxel `.vox` files that the renderer can load for vehicle parts. Install it separately; this repository does not include or redistribute the Kenney Shape ZIP, executable, or other application files.
- An optional MelonLoader probe lives in `adapters/kenney-shape-probe/`. The local Kenney Shape package is Unity 2021.3.45f2 with Mono managed assemblies, which makes the adapter plausible; it has not yet been bootstrapped with MelonLoader. The probe only reports loaded assemblies and candidate export/save methods. It does not patch or automate the application yet. The adapter is optional and is not required for the image preparation command or for using exported `.vox` files.

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

Run checks with:

```sh
python -m unittest discover -s tests -v
```

## Asset handoff to BN

For a vehicle part, export an authored `.vox` from Kenney Shape and copy it to BN's `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. The renderer uses that model for the matching vehicle part; missing assets continue to use its normal renderer path. Keep source projects and licenses with the asset during development. Do not put editor applications or their archives in the BN game data directory.

For character assets, first use the four-view sheet to generate a candidate model, then inspect its OBJ and palette before choosing an import or conversion format. Do not treat a generated model as renderer-ready until it has been checked in-game from all four directions.

## Project structure

- `src/sprite_pipeline/`: command-line preparation and validation.
- `tests/`: deterministic unit tests.
- `docs/roadmap.md`: integration milestones and acceptance checks.
- `docs/tools.md`: tool boundaries, installation links, and licensing notes.
