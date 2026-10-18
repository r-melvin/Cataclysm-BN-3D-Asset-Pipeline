# External tool boundaries

## Kenney Shape

Website: https://kenney.nl/tools/kenney-shape  
Documentation: https://kenney.nl/knowledge-base/tools/kenney-shape-documentation

Kenney Shape is optional and must be installed by the user. Do not check in its ZIP, binaries, Unity data, or bundled samples. Kenney states that creations can be used in production and documents VOX export; verify any palette or source-art licenses separately.

The locally provided application is Unity 2021.3.45f2 and includes `MonoBleedingEdge` plus `Managed/Assembly-CSharp.dll`. MelonLoader's current modder guide recommends `.NET Standard 2.1` for Mono games on Unity 2021.2 or newer, matching this app's Unity version. A local-only probe is scaffolded at `adapters/kenney-shape-probe/`; it checks for the expected game assembly and logs candidate method signatures without changing the app. It has not been compiled against MelonLoader or run inside Kenney Shape; bootstrap and export hooks remain unverified. Any adapter must be optional, local-only, and isolated from the BN game runtime. If automation is not reliable, provide documented UI steps.

## 2d-to-3d-voxelizer

Source: https://github.com/GazPrash/2d-to-3d-voxelizer  
License: MIT in the upstream repository; recheck upstream before copying or vendoring code.

The documented quad input order is `[LEFT, FRONT, RIGHT, BACK]`. Its published output is OBJ (with MTL/texture support in current releases). The core preparation tool creates the required input sheet, but generation currently remains an external application step. A future CLI adapter should use a pinned upstream revision and preserve license notices.

## Bright Nights renderer

The renderer consumes authored MagicaVoxel `.vox` files from its game-data asset directory. `sprite-pipeline inspect-vox` currently validates the renderer's single-model `SIZE`/`XYZI` subset and emits provenance metadata; it does not convert voxelizer OBJ output. This repository may emit manifests and conversion products for BN, but must not change game rules. Renderer-side integration remains in the separate BN source repository.
