package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
)

func TestWriteMagicaVoxelNormalizesCoordinatesAndPreservesPalette(t *testing.T) {
	voxels := NewVoxelMap(2)
	voxels.Set(Vector3{x: 0, y: 3, z: -2}, Voxel{
		Color:      RGB{r: 255, g: 0, b: 0},
		LeftColor:  RGB{r: 0, g: 255, b: 0},
		RightColor: RGB{r: 0, g: 0, b: 255},
	})
	voxels.Set(Vector3{x: 1, y: 2, z: 0}, Voxel{Color: RGB{r: 0, g: 0, b: 255}})

	var output bytes.Buffer
	if err := WriteMagicaVoxel(context.Background(), &output, voxels); err != nil {
		t.Fatalf("WriteMagicaVoxel returned error: %v", err)
	}
	data := output.Bytes()
	if string(data[:4]) != "VOX " || binary.LittleEndian.Uint32(data[4:8]) != 150 {
		t.Fatalf("unexpected VOX header: %q version %d", data[:4], binary.LittleEndian.Uint32(data[4:8]))
	}
	if string(data[20:24]) != "SIZE" {
		t.Fatalf("first chunk = %q, want SIZE", data[20:24])
	}
	if got := [3]uint32{
		binary.LittleEndian.Uint32(data[32:36]),
		binary.LittleEndian.Uint32(data[36:40]),
		binary.LittleEndian.Uint32(data[40:44]),
	}; got != [3]uint32{3, 2, 2} {
		t.Fatalf("model dimensions = %v, want [3 2 2]", got)
	}
	if string(data[44:48]) != "XYZI" {
		t.Fatalf("second chunk = %q, want XYZI", data[44:48])
	}
	if got := binary.LittleEndian.Uint32(data[56:60]); got != 2 {
		t.Fatalf("voxel count = %d, want 2", got)
	}
	first_voxel := data[60:64]
	second_voxel := data[64:68]
	if !bytes.Equal(first_voxel, []byte{0, 0, 1, 2}) || !bytes.Equal(second_voxel, []byte{2, 1, 0, 1}) {
		t.Fatalf("normalized voxels = %v, %v", first_voxel, second_voxel)
	}
	if string(data[68:72]) != "RGBA" {
		t.Fatalf("third chunk = %q, want RGBA", data[68:72])
	}
	if got := data[80:88]; !bytes.Equal(got, []byte{0, 0, 255, 255, 255, 0, 0, 255}) {
		t.Fatalf("palette colors = %v, want blue then red RGBA", got)
	}
}

func TestWriteMagicaVoxelRejectsUnsupportedDimensions(t *testing.T) {
	voxels := NewVoxelMap(257)
	voxels.Set(Vector3{x: 0, y: 0, z: 0}, Voxel{Color: RGB{r: 1}})
	voxels.Set(Vector3{x: 0, y: 0, z: 256}, Voxel{Color: RGB{r: 1}})
	var output bytes.Buffer
	if err := WriteMagicaVoxel(context.Background(), &output, voxels); err == nil {
		t.Fatal("expected oversized model dimensions to be rejected")
	}
}
