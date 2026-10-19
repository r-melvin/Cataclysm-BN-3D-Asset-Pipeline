# Bright Nights 3D asset pipeline

`sprite-pipeline` is one Go command-line application for preparing directional sprite art and voxel assets for the Cataclysm: Bright Nights 3D renderer. It packs four sprite views, voxelizes them, validates `.vox` files, and automates Kenney Shape exports through subcommands.

## Install

Install [Go 1.23 or newer](https://go.dev/dl/), then:

```text
go install github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/cmd/sprite-pipeline@latest
sprite-pipeline --help
```

`go install` puts the binary in Go's bin folder (`go env GOPATH` + `/bin`). If `sprite-pipeline` is not found, add that folder to your `PATH` and reopen the terminal.

That is everything the image-to-VOX commands need. The optional `kenney-export` command additionally needs the [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0), [Kenney Shape](https://kenney.nl/tools/kenney-shape) and [MelonLoader](https://github.com/LavaGang/MelonLoader); see [Export from Kenney Shape](#export-from-kenney-shape).

To build from a clone instead: `go build ./cmd/sprite-pipeline` from the repository root.

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

Kenney Shape is a separate Unity application; this command automates it and does not touch Bright Nights. Setup, once:

1. Install the [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) and check `dotnet --version` works.
2. Install [Kenney Shape](https://kenney.nl/tools/kenney-shape), install [MelonLoader](https://github.com/LavaGang/MelonLoader) into it, and launch Kenney Shape once through MelonLoader so it creates its `MelonLoader` and `Mods` folders.

Then pass the Kenney Shape install folder (or the macOS `.app` bundle, or a folder containing it) as `--game-dir`:

```text
sprite-pipeline kenney-export --game-dir "path/to/Kenney Shape" --project "path/to/prop.kenshape" --output "build/prop.vox" --asset-kind vehicle-part --asset-id prop
```

The command builds its internal adapter against the local MelonLoader assemblies, starts Unity with `-batchmode -nographics`, loads the project, calls Kenney's voxel writer directly, validates the output, and writes a manifest plus Unity log. This automates loading and exporting an existing `.kenshape`; it does not create geometry or convert sprite images into Kenney projects. Use `--force` to replace an existing output and manifest.

The CLI has platform-specific launch and MelonLoader paths for Linux, Windows, and macOS, with path-resolution tests on all three. The actual Kenney batch export was smoke-tested on Linux Unity 2021.3.45f2 Mono. Windows/macOS application-level export has not been smoke-tested because no Kenney installations were available on those systems. On macOS, compatibility depends on matching the Kenney app and MelonLoader architectures.

## Help and troubleshooting

Show all commands, or options for any command:

```text
sprite-pipeline --help
sprite-pipeline voxelize --help
sprite-pipeline kenney-export --help
sprite-pipeline prepare-four-view --help
sprite-pipeline inspect-vox --help
```

- If `sprite-pipeline`, `go` or `dotnet` is not recognized, install the matching tool and reopen the terminal so its `PATH` updates. .NET is needed only for `kenney-export`.
- If the output VOX or manifest already exists, pass `--force` to `voxelize` or `kenney-export` to replace it.
- If `kenney-export` cannot find Kenney Shape, verify `--game-dir` points to its install folder or macOS app bundle and that launching Kenney Shape once through MelonLoader created its `MelonLoader` and `Mods` folders. Confirm the `dotnet` command works.
- Kenney export writes a `.kenney.log` file beside the requested VOX. If the export fails, inspect that log and verify that the `.kenshape` project opens in the installed Kenney Shape version.
- Linux Kenney export was tested with Unity 2021.3.45f2 Mono. Windows/macOS paths have unit tests and native Go builds, but application-level Kenney export remains unverified there. On macOS, Kenney Shape and MelonLoader must use compatible architectures.

## Renderer handoff

Vehicle-part files belong at `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. Optional item files belong at `data/raw/r3d/items/<item-type-id>.vox`; enable the in-game **R3D voxel items** option to show them. Character models and OBJ import remain future work. Keep source projects and licenses with generated assets during development, and do not put editor applications or archives in BN game data.

Kenney Shape and MelonLoader are installed separately. This repository does not ship their ZIP files, binaries, Unity data, or bundled sample projects. The voxelizer backend is adapted from [GazPrash/2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer); its MIT attribution is retained in `third_party/licenses/2d-to-3d-voxelizer/`.

## Developer checks

```text
go test ./...
go vet ./...
gofmt -w cmd internal
```
