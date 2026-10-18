# Kenney Shape local probe

This optional MelonLoader mod is an isolated Kenney Shape export helper. Press **F8** in Kenney Shape to invoke its `ExportVoxel.Export()` method and redirect the app's VOX save dialog to a unique timestamped file. The output directory defaults to `Exports/` beside the game executable; set `BN3D_VOX_OUTPUT_DIRECTORY` to choose another location. The adapter does not hook or load into Bright Nights.

The tested Kenney Shape copy is Unity `2021.3.45f2`, uses `MonoBleedingEdge`, and reports MelonLoader runtime `net35`. The mod targets `.NET Standard 2.1` and references the app's Mono-compatible `MelonLoader.dll` and `0Harmony.dll` from `MelonLoader/net35`. Static inspection of the installed game assemblies confirmed the parameterless `ExportVoxel.Export()` entry point and the `Crosstales.FB.FileBrowser.SaveFile(string, string, string, ExtensionFilter[])` call it uses. The Harmony prefix redirects that save call only while the adapter-triggered export is running.

The adapter has been built without warnings against the local `net35` assemblies. MelonLoader's native bootstrap identified Kenney Shape correctly and reached plugin loading in a disposable app copy, but stopped before its mod-loading stage when Unity reported a `0 x 0` desktop. This environment has no usable graphical display, so mod loading, the F8 interaction, and the resulting file have not been confirmed end-to-end. Treat the export workflow as a local prototype and review the app log/output before relying on it.

Build after separately installing MelonLoader for the local Kenney Shape copy and locating its `MelonLoader.dll`:

```sh
MELONLOADER_DIR="/path/to/Kenney Shape/MelonLoader/net35" \
  dotnet build adapters/kenney-shape-probe/KenneyShapeProbe.csproj
```

Install the resulting `KenneyShapeProbe.dll` only in that Kenney Shape installation's `Mods` folder. Launch Kenney Shape using the MelonLoader instructions for your platform, load the project, and press F8 to export. Keep Kenney Shape and MelonLoader binaries outside this repository. If the hook fails, use Kenney Shape's normal export UI.

References:

- MelonLoader modder guide: https://github.com/LavaGang/MelonWiki/blob/master/docs/modders/quickstart.md
- MelonLoader getting started: https://github.com/LavaGang/MelonWiki/blob/master/docs/gettingstarted.md
- Kenney Shape: https://kenney.nl/tools/kenney-shape
