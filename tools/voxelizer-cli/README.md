# Headless quad voxelizer

This CLI adapts the Go generation backend from [GazPrash/2d-to-3d-voxelizer](https://github.com/GazPrash/2d-to-3d-voxelizer) to write a single-model MagicaVoxel `.vox` file. It consumes the four-view sheet produced by `sprite-pipeline prepare-four-view` in `[LEFT, FRONT, RIGHT, BACK]` order.

Build and run from this directory with Go 1.23 or newer:

```sh
go test ./...
mkdir -p ../../build
go build -o ../../build/bn3d-voxelizer ./cmd/voxelizer
../../build/bn3d-voxelizer \
  --input ../../build/wheel-quad.png \
  --output ../../build/wheel.vox \
  --voxel-scale 1
```

`--voxel-scale` accepts values from 1 to 1024. Larger values reduce the generated size. The writer normalizes upstream axes to the BN renderer convention (`x` = front/back, `y` = horizontal, `z` = vertical), limits dimensions to 256 voxels and output count to one million, and uses a deterministic palette.

MagicaVoxel and the current renderer assign one palette color to each occupied voxel. The upstream generator keeps distinct side colors for its OBJ mesh output, which cannot be represented in this `.vox` file. This CLI stores each voxel's front/back color, so side faces reuse it. Alpha is used to determine occupancy; partial alpha is treated as solid wherever the upstream generator considers it occupied. Use the upstream GUI OBJ export when separate face colors are required.

The adapted backend and its license/attribution are recorded in `../../third_party/licenses/2d-to-3d-voxelizer/`. The Go module fetches `github.com/nfnt/resize` at the version pinned in `go.mod`; neither dependency is bundled as a binary.
