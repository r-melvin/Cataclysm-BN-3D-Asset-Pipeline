# Bright Nights 3D asset pipeline

A separate project for preparing directional sprite art and voxel assets for the Cataclysm: Bright Nights 3D renderer. It does not modify BN game rules or ship external editor applications.

## Workflows

The `sprite-pipeline` Python CLI prepares four-view sheets and validates renderer-compatible `.vox` files. The standalone Go CLI in `tools/voxelizer-cli/` converts the voxelizer's `[LEFT, FRONT, RIGHT, BACK]` quad sheet into a MagicaVoxel file. It is adapted from [GazPrash/2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer); its MIT attribution and license are retained in `third_party/licenses/2d-to-3d-voxelizer/`.

[Kenney Shape](https://kenney.nl/tools/kenney-shape) can author and export `.kenshape` projects. On the tested Linux Mono build, `sprite-pipeline kenney-export` launches the user's installed application under Unity `-batchmode -nographics`, loads a project, and writes a renderer-ready `.vox` using Kenney's own `VoxFileIO.SaveFile` method. This headless route requires a local MelonLoader installation and .NET SDK to compile the adapter. It does not distribute Kenney Shape or MelonLoader, and it does not author models or convert PNGs into Kenney projects. The adapter stays isolated from BN.

### Install the Python CLI

Requires Python 3.11+ and Pillow:

```sh
python -m pip install -e .
```

### Convert four sprite views with the voxelizer

```sh
sprite-pipeline prepare-four-view \
  --left path/to/left.png --front path/to/front.png \
  --right path/to/right.png --back path/to/back.png \
  --output build/vehicle-part-quad.png

go -C tools/voxelizer-cli test ./...
go -C tools/voxelizer-cli build -o ../../build/bn3d-voxelizer ./cmd/voxelizer
./build/bn3d-voxelizer --input build/vehicle-part-quad.png \
  --output build/vehicle-part.vox
sprite-pipeline inspect-vox --input build/vehicle-part.vox \
  --asset-kind vehicle-part --asset-id vehicle-part
```

The packer requires four same-size views and writes them in `[LEFT, FRONT, RIGHT, BACK]` order. The adapted upstream quad generator writes `.vox` directly instead of routing through OBJ. The result retains voxel occupancy and front/back colors; separate side-face colors from the source mesh generator cannot be represented by one palette index per voxel and are lost. The CLI enforces renderer limits of 256 voxels per axis and 1,000,000 occupied cells. `--voxel-scale` (1–1024) can reduce the generated model.

### Headlessly export a Kenney Shape project

Install Kenney Shape and MelonLoader yourself, then launch the app once with MelonLoader enabled so it creates its local folders. Build and export with:

```sh
sprite-pipeline kenney-export \
  --game-dir /path/to/Kenney\ Shape \
  --project /path/to/prop.kenshape \
  --output build/prop.vox \
  --asset-kind vehicle-part --asset-id prop
```

The command builds the adapter against the installed MelonLoader assembly, copies it into that installation's `Mods` directory, starts Kenney Shape with `-batchmode -nographics`, then validates the result and writes `build/prop.manifest.json`. The Unity log is saved as `build/prop.kenney.log`. Use `--force` to replace an existing output and manifest. This path requires a Linux x86-64 Unity Mono installation; the installed project is loaded through Kenney's `Project.LoadPath(string)` API. Since export uses the app's VOX generator and writer directly, no display, camera render texture, or save dialog is used. A project's authored data is still required—the CLI's parameters control input/output paths and the asset manifest, not shape generation.

Validate a `.vox` produced elsewhere with `sprite-pipeline inspect-vox --input path/to/model.vox --asset-kind vehicle-part`. The validator rejects malformed coordinates and multi-model scene files unsupported by the current renderer.

Run local checks with:

```sh
python -m unittest discover -s tests -v
go -C tools/voxelizer-cli test ./...
go -C tools/voxelizer-cli vet ./...
```

### Handoff to BN

Vehicle-part files belong at `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. Optional item files belong at `data/raw/r3d/items/<item-type-id>.vox`; enable the in-game **R3D voxel items** option to show them. Character models and OBJ import are future work. Keep source projects and licenses with generated assets during development; do not put editor applications or archives in BN game data.

## External tools and licensing

See [`docs/tools.md`](docs/tools.md) for official project links, tool boundaries, and upstream attribution. Kenney Shape and MelonLoader are optional separately installed dependencies; this repository contains neither application binaries nor their archives.
