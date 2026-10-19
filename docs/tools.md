# External tool boundaries

## Kenney Shape

Website: https://kenney.nl/tools/kenney-shape  
Documentation: https://kenney.nl/knowledge-base/tools/kenney-shape-documentation

Kenney Shape is optional and must be installed by the user. Do not check in its ZIP, binaries, Unity data, or bundled samples. Verify palette and source-art licenses separately.

The tested local Linux build uses Unity 2021.3.45f2 and Mono. Static inspection and a real `-batchmode -nographics` run verified that its `Project.LoadPath(string)` loads `.kenshape` project files; the `ExportVoxel` class generates the voxel data and Kenney's `VoxFileIO.SaveFile(string, VoxFile)` writes it without camera rendering. The adapter invokes those paths in batch mode and bypasses the GUI save dialog. The game binary itself and its local MelonLoader installation are runtime prerequisites, never repository contents. The CLI does not create or edit Kenney project geometry.

The adapter also retains an optional F8 path for exporting the active model in a GUI session. Both paths only handle output file generation; keep the mod isolated from BN and install it only in Kenney Shape.

## 2d-to-3d-voxelizer

Source: https://github.com/GazPrash/2d-to-3d-voxelizer  
License: MIT in the upstream repository; recheck upstream before copying or vendoring code.

The documented quad input order is `[LEFT, FRONT, RIGHT, BACK]`. The upstream GUI exports OBJ with MTL/texture support. This repository includes an internal headless backend adapted from the upstream Go generator at revision `65ccdf72b564567925a7ff3557960f08b4daed54`; the public `sprite-pipeline voxelize` command packs/loads the input, invokes the backend, validates its output, and writes the asset manifest. The original MIT license and attribution are preserved in `third_party/licenses/2d-to-3d-voxelizer/`.

The conversion retains voxel occupancy and the front/back color assigned to each voxel. The source generator stores separate side-face colors (and top/bottom in six-sided mode) for mesh export, but MagicaVoxel and the current BN `.vox` renderer store one palette index per occupied voxel. The CLI therefore loses per-face material/color fidelity: side faces render with the voxel's front/back color. It is limited to quad sheets, applies the BN renderer axis convention (`x` front/back, `y` horizontal, `z` vertical), and enforces the 256-voxel axis and 1,000,000-cell bounds. Choose upstream OBJ export when individual face colors matter.

## Bright Nights renderer handoff

The renderer consumes authored MagicaVoxel `.vox` files from its game-data asset directory. Vehicle parts use `data/raw/r3d/vehicle_parts/<native-part-id>.vox`. Optional item assets use `data/raw/r3d/items/<item-type-id>.vox` and require the in-game **R3D voxel items** option. `sprite-pipeline inspect-vox` validates the single-model `SIZE`/`XYZI` subset and emits provenance metadata; it does not convert voxelizer OBJ output. Character model support and OBJ import remain future work. This repository emits manifests and conversion products for BN but does not modify game rules.
