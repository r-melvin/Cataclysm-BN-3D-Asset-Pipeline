# Bright Nights 3D asset pipeline

One application, `sprite-pipeline`, prepares directional sprite art and voxel assets for the Cataclysm: Bright Nights 3D renderer. Its subcommands provide the stages for packing, voxelizing, validating, and exporting. The Go voxelizer and MelonLoader adapter are internal backends; users run all workflows through `sprite-pipeline`.

## Install

Requires Python 3.11+ and Pillow. The sprite-to-voxel stage also needs Go 1.23+. The Kenney Shape export stage needs the separately installed Linux Mono app, MelonLoader, and a .NET SDK.

```sh
python -m pip install -e .
```

## Convert directional sprites to VOX

Provide four equal-sized views in the documented left/front/right/back order. The single command packs them, runs the internal voxelizer backend, validates the generated model, and writes an asset manifest:

```sh
sprite-pipeline voxelize \
  --left path/to/left.png --front path/to/front.png \
  --right path/to/right.png --back path/to/back.png \
  --output build/wheel.vox \
  --asset-kind vehicle-part --asset-id wheel
```

An already packed PNG/JPEG sheet can be passed with `--sheet` instead. Its views must be arranged horizontally as `[LEFT, FRONT, RIGHT, BACK]`:

```sh
sprite-pipeline voxelize --sheet build/wheel-quad.png \
  --output build/wheel.vox --asset-kind vehicle-part --asset-id wheel
```

`--voxel-scale` accepts values from 1 to 1024; larger values reduce the generated size. The adapted upstream quad generator writes `.vox` directly instead of routing through OBJ. Output retains voxel occupancy and front/back colors, but per-side face colors are lost because MagicaVoxel stores one color per voxel. Models are limited to 256 voxels per axis and 1,000,000 occupied cells.

Use stage commands when an intermediate is useful:

```sh
sprite-pipeline prepare-four-view \
  --left path/to/left.png --front path/to/front.png \
  --right path/to/right.png --back path/to/back.png \
  --output build/wheel-quad.png
sprite-pipeline inspect-vox --input build/wheel.vox \
  --asset-kind vehicle-part --asset-id wheel
```

## Export a Kenney Shape project

[Kenney Shape](https://kenney.nl/tools/kenney-shape) is an optional authoring tool. Install it yourself, install MelonLoader in its Unity 2021.3.45f2 Mono build, and launch once so MelonLoader creates its local folders. Then use the same `sprite-pipeline` application:

```sh
sprite-pipeline kenney-export \
  --game-dir /path/to/Kenney\ Shape \
  --project /path/to/prop.kenshape \
  --output build/prop.vox \
  --asset-kind vehicle-part --asset-id prop
```

The command builds and installs the internal adapter, launches Kenney Shape with Unity `-batchmode -nographics`, loads the project, and calls Kenney's voxel writer directly. It then validates the `.vox` and writes a manifest. The Unity log is saved beside the output. Use `--force` to replace an existing output and manifest.

This path automates loading and exporting an existing `.kenshape`; it does not author geometry or convert sprite images into Kenney projects. It is currently tested on Linux x86-64 with Unity Mono. It does not require camera rendering, a display, or the GUI save dialog.

## Renderer handoff

Vehicle-part files belong at `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. Optional item files belong at `data/raw/r3d/items/<item-type-id>.vox`; enable the in-game **R3D voxel items** option to show them. Character models and OBJ import remain future work. Keep source projects and licenses with generated assets during development, and do not put editor applications or archives in BN game data.

Kenney Shape and MelonLoader are optional dependencies installed separately. This repository does not ship their ZIP files, binaries, Unity data, or bundled sample projects. The Go backend is adapted from [GazPrash/2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer); its MIT attribution is retained in `third_party/licenses/2d-to-3d-voxelizer/`.

## Developer checks

```sh
PYTHONPATH=src python -m unittest discover -s tests -v
go -C tools/voxelizer-backend test ./...
go -C tools/voxelizer-backend vet ./...
```
