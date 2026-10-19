# Bright Nights 3D asset pipeline

`sprite-pipeline` is one Go command-line application for preparing directional sprite art and voxel assets for the Cataclysm: Bright Nights 3D renderer. It packs four sprite views, voxelizes them, validates `.vox` files, and automates Kenney Shape exports through subcommands. Go and the MelonLoader bridge are internal implementation details; users run `sprite-pipeline` for each workflow.

## Install

Install [Go 1.23 or newer](https://go.dev/dl/), then build the executable from this repository's root.

On Linux or macOS:

```text
go build -o sprite-pipeline ./cmd/sprite-pipeline
```

On Windows PowerShell:

```text
go build -o sprite-pipeline.exe ./cmd/sprite-pipeline
```

The built Go CLI uses no Python installation or POSIX shell features. CI runs native tests and builds on Linux, macOS, and Windows.

## Convert directional sprites to VOX

Pass four equal-sized view images in left/front/right/back order. The command packs them, invokes the internal voxelizer, validates the `.vox`, and writes a manifest:

```text
sprite-pipeline voxelize --left "path/to/left.png" --front "path/to/front.png" --right "path/to/right.png" --back "path/to/back.png" --output "build/wheel.vox" --asset-kind vehicle-part --asset-id wheel
```

An already packed PNG/JPEG sheet can be passed with `--sheet`. Its views must run horizontally in `[LEFT, FRONT, RIGHT, BACK]` order:

```text
sprite-pipeline voxelize --sheet "build/wheel-quad.png" --output "build/wheel.vox" --asset-kind vehicle-part --asset-id wheel
```

`--voxel-scale` accepts values from 1 to 1024; larger values reduce the generated size. The adapted upstream quad generator writes `.vox` directly instead of routing through OBJ. Output retains voxel occupancy and front/back colors, but per-side face colors are lost because MagicaVoxel stores one color per voxel. Models are limited to 256 voxels per axis and 1,000,000 occupied cells. Use `--force` to replace an existing model and manifest.

Use the packing stage separately when you need to inspect or keep the intermediate sheet:

```text
sprite-pipeline prepare-four-view --left "left.png" --front "front.png" --right "right.png" --back "back.png" --output "build/wheel-quad.png"
```

Validate an existing model and write its provenance manifest:

```text
sprite-pipeline inspect-vox --input "build/wheel.vox" --asset-kind vehicle-part --asset-id wheel
```

## Export from Kenney Shape

[Kenney Shape](https://kenney.nl/tools/kenney-shape) is an optional authoring tool. Install the app and [MelonLoader](https://github.com/LavaGang/MelonLoader) for your operating system, then launch Kenney Shape once through MelonLoader so it creates its folders. Run the export with the same Go executable:

The `kenney-export` subcommand also needs the .NET 8 SDK (`dotnet` on `PATH`) to build its small adapter against the MelonLoader assemblies installed with the game. Go builds, voxelization, packing, and VOX inspection do not need .NET or Kenney Shape.

```text
sprite-pipeline kenney-export --game-dir "path to Kenney Shape" --project "path to prop.kenshape" --output "build/prop.vox" --asset-kind vehicle-part --asset-id prop
```

The command builds its internal adapter against the local MelonLoader assemblies, starts Unity with `-batchmode -nographics`, loads the project, calls Kenney's voxel writer directly, validates the output, and writes a manifest plus Unity log. This automates loading and exporting an existing `.kenshape`; it does not create geometry or convert sprite images into Kenney projects. Use `--force` to replace an existing output and manifest.

The CLI has platform-specific launch and MelonLoader paths for Linux, Windows, and macOS, with path-resolution tests on all three. The actual Kenney batch export was smoke-tested on Linux Unity 2021.3.45f2 Mono. Windows/macOS application-level export has not been smoke-tested because no Kenney installations were available on those systems. On macOS, compatibility depends on matching the Kenney app and MelonLoader architectures.

## Run the command

On Linux or macOS:

```text
./sprite-pipeline --help
```

On Windows PowerShell:

```text
.\sprite-pipeline.exe --help
```

## Renderer handoff

Vehicle-part files belong at `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. Optional item files belong at `data/raw/r3d/items/<item-type-id>.vox`; enable the in-game **R3D voxel items** option to show them. Character models and OBJ import remain future work. Keep source projects and licenses with generated assets during development, and do not put editor applications or archives in BN game data.

Kenney Shape and MelonLoader are installed separately. This repository does not ship their ZIP files, binaries, Unity data, or bundled sample projects. The voxelizer backend is adapted from [GazPrash/2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer); its MIT attribution is retained in `third_party/licenses/2d-to-3d-voxelizer/`.

## Developer checks

```text
go test ./...
go vet ./...
gofmt -w cmd internal
```
