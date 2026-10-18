# External tool boundaries

## Kenney Shape

Website: https://kenney.nl/tools/kenney-shape  
Documentation: https://kenney.nl/knowledge-base/tools/kenney-shape-documentation

Kenney Shape is optional and must be installed by the user. Do not check in its ZIP, binaries, Unity data, or bundled samples. Kenney states that creations can be used in production and documents VOX export; verify any palette or source-art licenses separately.

The locally provided application is Unity 2021.3.45f2 and includes `MonoBleedingEdge` plus `Managed/Assembly-CSharp.dll`. The optional `adapters/kenney-shape-probe/` helper targets `.NET Standard 2.1` and compiles against the app's `MelonLoader/net35` assemblies. Static inspection confirmed its `ExportVoxel.Export()` entry point and VOX save-dialog method. Pressing F8 invokes this path and redirects the save dialog to a generated output name. The native loader bootstrap was exercised on a disposable app copy and identified the game, but stopped before mod loading when Unity reported a `0 x 0` desktop. GUI runtime loading and a completed export remain unverified due to the unavailable display. Keep the adapter optional and isolated from the BN game runtime. If automation is not reliable, use the documented UI path.

## 2d-to-3d-voxelizer

Source: https://github.com/GazPrash/2d-to-3d-voxelizer  
License: MIT in the upstream repository; recheck upstream before copying or vendoring code.

The documented quad input order is `[LEFT, FRONT, RIGHT, BACK]`. Its published output is OBJ (with MTL/texture support in current releases). The core preparation tool creates the required input sheet, but generation currently remains an external application step. A future CLI adapter should use a pinned upstream revision and preserve license notices.

## Bright Nights renderer

The renderer consumes authored MagicaVoxel `.vox` files from its game-data asset directory. Vehicle parts use `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. Optional item assets use `data/raw/r3d/items/<item-type-id>.vox` and require the in-game **R3D voxel items** option. `sprite-pipeline inspect-vox` validates the renderer's single-model `SIZE`/`XYZI` subset and emits provenance metadata; it does not convert voxelizer OBJ output. Character model support and OBJ import remain future work. This repository may emit manifests and conversion products for BN, but must not change game rules. Renderer-side integration remains in the separate BN source repository.
