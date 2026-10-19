# Roadmap

## 1. Repeatable input and asset validation (implemented)

- Accept four PNG files in explicit left/front/right/back order.
- Require equal dimensions and preserve source pixels, transparency, and palette appearance.
- Emit the voxelizer-compatible horizontal sheet and a provenance manifest with SHA-256 hashes.
- Run the pinned upstream quad generator through the internal Go backend called by `sprite-pipeline voxelize`; write renderer-compatible `.vox` output and document the single-color-per-voxel face-color loss.
- Inspect single-model MagicaVoxel files against the current renderer's `SIZE`/`XYZI` subset.
- Reject unsupported multi-model scenes, invalid palette indices, duplicate voxels, and out-of-range coordinates; emit asset manifests for vehicle parts and characters.

## 2. Kenney Shape export helper (headless path implemented; cross-platform runtime checks pending)

- Confirm the installed Kenney Shape version and whether it uses Unity Mono or IL2CPP.
- Inspect only the user's separately installed application; never include its ZIP, executable, assemblies, or assets in this repository.
- `sprite-pipeline kenney-export` embeds and builds the `KenneyShapeProbe` against the user's MelonLoader assemblies, loads `.kenshape` through the installed `Project.LoadPath` API, and calls Kenney's VOX writer directly in batch mode.
- Linux Unity 2021.3.45f2 Mono load/export is validated. Windows and macOS installation path/bootstrap resolution has unit coverage and native CI builds; repeat runtime smoke tests when compatible Kenney/MelonLoader installs are available on those systems.
- Keep MelonLoader, Kenney Shape, and its project files as user-installed dependencies; never include their binaries or assets.

## 3. Vehicle model workflow

- Create and visually inspect a wheel and one body part in Kenney Shape.
- Export `.vox`, validate it with `sprite-pipeline inspect-vox`, and copy it to the BN vehicle-part asset directory under its native part ID.
- Compare in a real BN vehicle scene across yaw, open/broken states, lighting, and adjacent part configurations.
- Record palette, scale normalization, pivot, source, and license beside each model.

Vehicle asset handoff path: `data/raw/r3d/vehicle_parts/<native-part-id>.vox`.

## 3a. Optional voxel item workflow

- Create a single-model `.vox` file named for the BN item type ID.
- Validate it with `sprite-pipeline inspect-vox` and place it at `data/raw/r3d/items/<item-type-id>.vox`.
- Enable the renderer's **R3D voxel items** option to use these optional assets; missing files continue to use the normal renderer path.

## 4. Character model workflow (future)

- Prepare and validate a four-view character sheet.
- Generate `.vox` with `sprite-pipeline voxelize` and assess its geometry/palette in the renderer. Use the upstream GUI OBJ output when per-face colors are needed.
- Improve the `.vox` boundary or renderer format if side textures must be preserved. Per-face color support and renderer-ready character assets remain future work.
- Preserve BN equipment layers and sprite fallback; verify facing selection against camera-relative travel.

## 5. End-to-end orchestration (implemented)

- The Go `sprite-pipeline` application exposes voxelization, validation, packing, and Kenney export as subcommands. It invokes only the internal Go generator and the user's installed Kenney/MelonLoader.
- Package only generated assets, manifests, and conversion settings; exclude saves and game data unless explicitly required for a debugging capture.
- Add deterministic validation and a small reference asset suite before considering batch conversion complete.
