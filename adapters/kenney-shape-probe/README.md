# Kenney Shape local probe

This optional MelonLoader mod checks whether the user's local Kenney Shape installation can be instrumented and logs loaded managed assembly names plus methods whose names suggest save/export/voxel functionality. It does not patch methods, write project files, or automate exports yet.

The provided Kenney Shape package was inspected locally: it contains Unity `2021.3.45f2`, `MonoBleedingEdge`, and `Managed/Assembly-CSharp.dll`. That makes MelonLoader's Mono support a plausible path, but the probe has not yet been compiled against or run with an installed MelonLoader. The tool application itself is not included in this repository.

Build only after installing MelonLoader for the local Kenney Shape copy and locating its `MelonLoader.dll`:

```sh
MELONLOADER_DIR="/path/to/Kenney Shape/MelonLoader" \
  dotnet build adapters/kenney-shape-probe/KenneyShapeProbe.csproj
```

Install the resulting `KenneyShapeProbe.dll` only in that Kenney Shape installation's `Mods` folder. Review the probe's log before writing any patches. If MelonLoader cannot bootstrap against this Unity build, use Kenney Shape's documented manual export path.
