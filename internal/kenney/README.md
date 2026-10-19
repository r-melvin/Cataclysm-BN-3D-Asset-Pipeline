# Internal Kenney Shape adapter

This adapter is built by the Go `sprite-pipeline kenney-export` command and installed into the user's local Kenney Shape `Mods` directory for the duration of the export. It must only be loaded by Kenney Shape, never by BN. Application, MelonLoader, and Unity files are user-installed and are not included in this repository.

## Headless export

Kenney Shape is the separate Unity application automated here; it is not Cataclysm: Bright Nights. Install MelonLoader into Kenney Shape, launch it once through MelonLoader, and pass its install folder (or its `.app` bundle on macOS) as `--game-dir`. The CLI uses that location to find Kenney Shape's MelonLoader assemblies when building the adapter and to launch the application for export. Then use the single `sprite-pipeline` application:

```text
sprite-pipeline kenney-export --game-dir "path/to/Kenney Shape" --project "path/to/model.kenshape" --output "build/model.vox" --asset-kind vehicle-part --asset-id model
```

The CLI builds the adapter against the local MelonLoader assemblies, loads an existing `.kenshape` project through Kenney's `Project.LoadPath(string)`, and invokes `VoxFileIO.SaveFile` directly. It launches Unity with `-batchmode -nographics`, then validates the model and writes a manifest and log. It does not author geometry or convert sprite images into Kenney projects.

The direct project-load and voxel-writer APIs were verified with Unity 2021.3.45f2 Mono on Linux. Windows uses MelonLoader's installed `version.dll` proxy; macOS uses its installed bootstrap dylib and Unity app bundle. Native Windows/macOS game launches are covered by path-resolution tests and CI builds, but this repository has no Kenney installations on those systems for runtime smoke tests. MelonLoader compatibility depends on the user's app architecture and loader installation.

The adapter retains F8 for GUI exports. It uses the same direct voxel writer, saving to `BN3D_VOX_OUTPUT_DIRECTORY` or an `Exports` folder under the Kenney Shape install folder.

Do not ship or commit the Kenney Shape ZIP, executable, Unity data, sample projects, MelonLoader, or runtime logs. If Kenney changes its internal API, update the compatibility checks and repeat the batch export smoke test.
