# Roadmap

## 1. Repeatable input and asset validation (implemented)

- Accept four PNG files in explicit left/front/right/back order.
- Require equal dimensions and preserve source pixels, transparency, and palette appearance.
- Emit the voxelizer-compatible horizontal sheet and a provenance manifest with SHA-256 hashes.
- Run the pinned upstream quad generator through a headless Go CLI and write renderer-compatible `.vox` output; document the single-color-per-voxel face-color loss.
- Inspect single-model MagicaVoxel files against the current renderer's `SIZE`/`XYZI` subset.
- Reject unsupported multi-model scenes, invalid palette indices, duplicate voxels, and out-of-range coordinates; emit asset manifests for vehicle parts and characters.

## 2. Kenney Shape export helper (prototype implemented; GUI validation pending)

- Confirm the installed Kenney Shape version and whether it uses Unity Mono or IL2CPP.
- Inspect only the user's separately installed application; never include its ZIP, executable, assemblies, or assets in this repository.
- The local-only `KenneyShapeProbe` builds against the installed `MelonLoader/net35` assemblies and Harmony. Static inspection confirmed the `ExportVoxel.Export()` entry point and VOX save dialog API. F8 calls the export method and redirects its save dialog to a timestamped file.
- Confirm mod loading, F8 interaction, and a valid exported file in a graphical Kenney Shape session. Keep the documented manual export path available if hooks are brittle or unavailable.
- Do not make a MelonLoader dependency part of the core pipeline.

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
- Generate `.vox` with the headless quad CLI and assess its geometry/palette in the renderer. Use the GUI OBJ output when per-face colors are needed.
- Improve the `.vox` boundary or renderer format if side textures must be preserved. Per-face color support and renderer-ready character assets remain future work.
- Preserve BN equipment layers and sprite fallback; verify facing selection against camera-relative travel.

## 5. End-to-end orchestration

- Add a manifest-driven command that invokes supported, locally installed tools without redistributing them.
- Package only generated assets, manifests, and conversion settings; exclude saves and game data unless explicitly required for a debugging capture.
- Add deterministic validation and a small reference asset suite before considering batch conversion complete.
