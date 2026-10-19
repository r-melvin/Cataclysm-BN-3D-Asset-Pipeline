package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
)

type voxelRecord struct {
	x, y, z int
	color   RGB
}

// WriteMagicaVoxel writes the generated voxel map as a single-model MagicaVoxel file.
// The format stores one palette color per occupied voxel, so this uses each voxel's
// front/rear Color field and cannot retain the upstream map's per-side face colors.
func WriteMagicaVoxel(ctx context.Context, writer io.Writer, voxels *VoxelMap) error {
	if voxels == nil || len(voxels.shards) == 0 {
		return fmt.Errorf("voxel map is empty")
	}

	records := make([]voxelRecord, 0)
	min_x, max_x := 0, 0
	min_y, max_y := 0, 0
	min_z, max_z := 0, 0
	first := true
	for x, shard := range voxels.shards {
		if err := ctx.Err(); err != nil {
			return err
		}
		for position, voxel := range shard.m {
			if first {
				min_x, max_x = position.z, position.z
				min_y, max_y = x, x
				min_z, max_z = position.y, position.y
				first = false
			} else {
				min_x = min(min_x, position.z)
				max_x = max(max_x, position.z)
				min_y = min(min_y, x)
				max_y = max(max_y, x)
				min_z = min(min_z, position.y)
				max_z = max(max_z, position.y)
			}
			// The upstream map uses x=image width, y=image rows (down), z=depth.
			// The renderer uses x=front/back, y=horizontal, z=height (up).
			records = append(records, voxelRecord{x: position.z, y: x, z: position.y, color: voxel.Color})
		}
	}
	if len(records) == 0 {
		return fmt.Errorf("voxel map contains no occupied voxels")
	}
	if len(records) > 1_000_000 {
		return fmt.Errorf("model contains %d voxels; the renderer supports at most 1000000", len(records))
	}

	size_x, size_y, size_z := max_x-min_x+1, max_y-min_y+1, max_z-min_z+1
	if size_x > 256 || size_y > 256 || size_z > 256 {
		return fmt.Errorf("model dimensions %dx%dx%d exceed the renderer's 256-voxel limit", size_x, size_y, size_z)
	}
	if uint64(len(records)) > uint64(size_x)*uint64(size_y)*uint64(size_z) {
		return fmt.Errorf("voxel count exceeds model volume")
	}

	colors := make([]RGB, 0)
	seen_colors := make(map[RGB]struct{})
	for _, record := range records {
		if _, exists := seen_colors[record.color]; !exists {
			seen_colors[record.color] = struct{}{}
			colors = append(colors, record.color)
		}
	}
	sort.Slice(colors, func(i, j int) bool {
		if colors[i].r != colors[j].r {
			return colors[i].r < colors[j].r
		}
		if colors[i].g != colors[j].g {
			return colors[i].g < colors[j].g
		}
		return colors[i].b < colors[j].b
	})
	if len(colors) > 255 {
		return fmt.Errorf("model uses %d colors; MagicaVoxel supports at most 255 opaque voxel colors", len(colors))
	}
	palette_indices := make(map[RGB]uint8, len(colors))
	for index, color := range colors {
		palette_indices[color] = uint8(index + 1)
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].x != records[j].x {
			return records[i].x < records[j].x
		}
		if records[i].y != records[j].y {
			return records[i].y < records[j].y
		}
		return records[i].z < records[j].z
	})

	size_chunk := make([]byte, 12)
	binary.LittleEndian.PutUint32(size_chunk[0:4], uint32(size_x))
	binary.LittleEndian.PutUint32(size_chunk[4:8], uint32(size_y))
	binary.LittleEndian.PutUint32(size_chunk[8:12], uint32(size_z))

	voxel_chunk := make([]byte, 4+len(records)*4)
	binary.LittleEndian.PutUint32(voxel_chunk[0:4], uint32(len(records)))
	for index, record := range records {
		if index%4096 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		offset := 4 + index*4
		voxel_chunk[offset] = uint8(record.x - min_x)
		voxel_chunk[offset+1] = uint8(record.y - min_y)
		voxel_chunk[offset+2] = uint8(record.z - min_z)
		voxel_chunk[offset+3] = palette_indices[record.color]
	}

	palette_chunk := make([]byte, 256*4)
	for index, color := range colors {
		offset := index * 4
		palette_chunk[offset] = color.r
		palette_chunk[offset+1] = color.g
		palette_chunk[offset+2] = color.b
		palette_chunk[offset+3] = 255
	}

	chunks := [][]byte{
		makeChunk("SIZE", size_chunk),
		makeChunk("XYZI", voxel_chunk),
		makeChunk("RGBA", palette_chunk),
	}
	children_size := 0
	for _, chunk := range chunks {
		children_size += len(chunk)
	}

	if _, err := io.WriteString(writer, "VOX "); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(150)); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, "MAIN"); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(0)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(children_size)); err != nil {
		return err
	}
	for _, chunk := range chunks {
		if _, err := writer.Write(chunk); err != nil {
			return err
		}
	}
	return nil
}

func makeChunk(name string, content []byte) []byte {
	chunk := make([]byte, 12+len(content))
	copy(chunk[:4], name)
	binary.LittleEndian.PutUint32(chunk[4:8], uint32(len(content)))
	copy(chunk[12:], content)
	return chunk
}
