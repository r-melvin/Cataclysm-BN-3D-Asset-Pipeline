# Roadmap

## 1. Repeatable input and asset validation (implemented)

- Accept four PNG files in explicit left/front/right/back order.
- Require equal dimensions and preserve source pixels, transparency, and palette appearance.
- Emit the voxelizer-compatible horizontal sheet and a provenance manifest with SHA-256 hashes.
- Inspect single-model MagicaVoxel files against the current renderer's `SIZE`/`XYZI` subset.
- Reject unsupported multi-model scenes, invalid palette indices, duplicate voxels, and out-of-range coordinates; emit asset manifests for vehicle parts and characters.

## 2. Kenney Shape automation feasibility (probe scaffolded; runtime validation pending)

- Confirm the installed Kenney Shape version and whether it uses Unity Mono or IL2CPP.
- Inspect only the user's separately installed application; never include its ZIP, executable, assemblies, or assets in this repository.
- Build and run the local-only `KenneyShapeProbe` against that runtime. It logs loaded assemblies and candidate method signatures without patching them. The local project targets `.NET Standard 2.1`, matching the documented MelonLoader target for Mono on Unity 2021.2 or newer.
- Determine whether the application exposes a safe project-open/export workflow. Prototype automation only after the probe identifies a stable path; keep a documented manual export path if hooks are brittle or unavailable.
- Do not make a MelonLoader dependency part of the core pipeline.

## 3. Vehicle model workflow

- Create and visually inspect a wheel and one body part in Kenney Shape.
- Export `.vox`, validate it with `sprite-pipeline inspect-vox`, and copy it to the BN vehicle-part asset directory under its native part ID.
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
