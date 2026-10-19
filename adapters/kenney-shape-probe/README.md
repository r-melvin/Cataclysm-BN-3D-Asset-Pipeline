# Kenney Shape MelonLoader adapter

This optional adapter lives outside BN and must only be installed in a user's Kenney Shape folder. It is built against that installation's `MelonLoader/net35/MelonLoader.dll`; proprietary app and loader binaries are not included here.

## Headless export

The supported automated mode is Linux x86-64 with the Unity Mono build. Install Kenney Shape and MelonLoader separately, launch once with MelonLoader so its bootstrap and folders are created, install the Python package from the repository root, then run:

```sh
sprite-pipeline kenney-export \
  --game-dir /path/to/Kenney\ Shape \
  --project /path/to/model.kenshape \
  --output build/model.vox \
  --asset-kind vehicle-part --asset-id model
```

The CLI builds and installs `KenneyShapeProbe.dll` in the application's `Mods` folder, sets input/output paths in the child process environment, and launches Unity with `-batchmode -nographics`. The adapter calls Kenney's `Project.LoadPath(string)`, reads the loaded project tile/palette data, uses Kenney's voxel generation helper, then calls its `VoxFileIO.SaveFile` API directly. It bypasses the file browser and does not require a window, rendered camera, or export dialog. The output is validated against the renderer's supported `.vox` subset and accompanied by a JSON manifest and Unity log.

The CLI automates loading and exporting an existing `.kenshape` project. It does not create project geometry or convert sprite images into Kenney projects. Tested app/API compatibility is Unity 2021.3.45f2 Mono; the adapter checks for the expected APIs at runtime and exits without exporting if they do not match.

## Optional GUI path

For a normal GUI session, F8 exports the active model to the directory set by `BN3D_VOX_OUTPUT_DIRECTORY` or to `Exports` under the game directory. This path uses the same direct voxel generator and writer as batch mode, without showing the save dialog. Headless automation should use the Python CLI above.

## Maintenance

Do not ship or commit the Kenney Shape ZIP, executable, Unity data, sample projects, MelonLoader, or runtime logs. Keep the adapter limited to the app's export bridge; it must never be loaded into BN or alter game behavior. If Kenney changes its internal API, update the compatibility check and repeat the actual batch export smoke test.
