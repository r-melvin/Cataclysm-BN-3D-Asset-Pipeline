# Roadmap

## 1. Repeatable four-view input preparation (implemented)

- Accept four PNG files in explicit left/front/right/back order.
- Require equal dimensions and preserve source pixels, transparency, and palette appearance.
- Emit the voxelizer-compatible horizontal sheet and a provenance manifest with SHA-256 hashes.

## 2. Kenney Shape automation feasibility (research next)

- Confirm the installed Kenney Shape version and whether it uses Unity Mono or IL2CPP.
- Inspect only the user's separately installed application; never include its ZIP, executable, assemblies, or assets in this repository.
- Build and run the local-only `KenneyShapeProbe` against that runtime. It currently logs assembly names and candidate save/export/voxel methods without patching them.
- Determine whether the application exposes a safe project-open/export workflow. Prototype automation only after the probe identifies a stable path; keep a documented manual export path if hooks are brittle or unavailable.
- Do not make a MelonLoader dependency part of the core pipeline.

## 3. Vehicle model workflow

- Create and visually inspect a wheel and one body part in Kenney Shape.
- Export `.vox`, validate it, and copy it to the BN vehicle-part asset directory under its native part ID.
- Compare in a real BN vehicle scene across yaw, open/broken states, lighting, and adjacent part configurations.
- Record palette, scale normalization, pivot, source, and license beside each model.

## 4. Character model workflow

- Prepare and validate a four-view character sheet.
- Generate an OBJ with 2d-to-3d-voxelizer and assess its geometry/palette and license.
- Select a maintainable OBJ-to-renderer asset boundary or adapt the voxelizer's exporter to `.vox`.
- Preserve BN equipment layers and sprite fallback; verify facing selection against camera-relative travel.

## 5. End-to-end orchestration

- Add a manifest-driven command that invokes supported, locally installed tools without redistributing them.
- Package only generated assets, manifests, and conversion settings; exclude saves and game data unless explicitly required for a debugging capture.
- Add deterministic validation and a small reference asset suite before considering batch conversion complete.
