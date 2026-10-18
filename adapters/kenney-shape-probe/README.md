# Kenney Shape local probe

This optional MelonLoader mod verifies the Kenney Shape installation path, logs loaded managed assemblies, and records full signatures for methods whose names suggest save, export, or voxel operations. It does not patch methods, write project files, or automate exports.

The local Kenney Shape copy is Unity `2021.3.45f2` and includes `MonoBleedingEdge` plus `Managed/Assembly-CSharp.dll`. MelonLoader's modder guide specifies `.NET Standard 2.1` for Mono games on Unity 2021.2 or newer, so the probe targets `netstandard2.1`. This establishes a framework-level match only; loader bootstrap and runtime compatibility in Kenney Shape remain unverified.

An offline string scan of `Assembly-CSharp.dll` exposes candidate names including `ExportVoxel`, `ExportModel`, `VoxFileModel`, `ProjectManager`, and `OBJExporter`. These names indicate relevant code paths may exist, but they do not establish method signatures, call requirements, or a stable automation hook; the runtime probe is intended to gather that missing reflection data.

Build after separately installing MelonLoader for the local Kenney Shape copy and locating its `MelonLoader.dll`:

```sh
MELONLOADER_DIR="/path/to/Kenney Shape/MelonLoader" \
  dotnet build adapters/kenney-shape-probe/KenneyShapeProbe.csproj
```

Install the resulting `KenneyShapeProbe.dll` only in that Kenney Shape installation's `Mods` folder. Review the log before designing a Harmony patch or export automation. Keep Kenney Shape and MelonLoader binaries outside this repository. If loader bootstrap proves unreliable, use Kenney Shape's documented manual export path.

References:

- MelonLoader modder guide: https://github.com/LavaGang/MelonWiki/blob/master/docs/modders/quickstart.md
- MelonLoader getting started: https://github.com/LavaGang/MelonWiki/blob/master/docs/gettingstarted.md
- Kenney Shape: https://kenney.nl/tools/kenney-shape
