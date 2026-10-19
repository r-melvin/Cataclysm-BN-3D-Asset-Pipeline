# Internal sprite-to-voxel backend

This Go backend is invoked by the public `sprite-pipeline voxelize` command. Users should use that command rather than building or launching the backend directly. It adapts the Go generation backend from [GazPrash/2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer) to write one MagicaVoxel `.vox` file.

Developers can run its package checks from this directory:

```sh
go test ./...
go vet ./...
go build ./cmd/voxelizer
```

The backend accepts a packed `[LEFT, FRONT, RIGHT, BACK]` PNG/JPEG sheet, applies BN renderer axis conventions and enforces the renderer model limits. The public CLI packs four directional PNGs when given them directly, invokes this backend, then validates the `.vox` and writes its provenance manifest.

The output keeps voxel occupancy and front/back voxel colors. The upstream OBJ path has separate side-face colors, which MagicaVoxel cannot represent as one palette entry per voxel. Source attribution and the upstream MIT license are preserved at `third_party/licenses/2d-to-3d-voxelizer/`.
