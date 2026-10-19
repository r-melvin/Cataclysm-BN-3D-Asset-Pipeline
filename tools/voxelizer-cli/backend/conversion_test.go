package backend

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestQuadConversionWritesRendererAlignedVox(t *testing.T) {
	input := image.NewRGBA(image.Rect(0, 0, 16, 4))
	views := []color.RGBA{
		{R: 0, G: 0, B: 255, A: 255},
		{R: 255, G: 0, B: 0, A: 255},
		{R: 0, G: 255, B: 0, A: 255},
		{R: 255, G: 255, B: 0, A: 255},
	}
	for panel, fill := range views {
		for x := panel * 4; x < (panel+1)*4; x++ {
			for y := 0; y < 4; y++ {
				input.SetRGBA(x, y, fill)
			}
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, input); err != nil {
		t.Fatalf("encode quad input: %v", err)
	}
	settings := Settings{Layout: "quad", Shape: "flat", VoxelScale: 1, FlatDepth: 1}
	voxels, _, err := ConvertTo3D(
		context.Background(), base64.StdEncoding.EncodeToString(encoded.Bytes()), settings,
	)
	if err != nil {
		t.Fatalf("ConvertTo3D returned error: %v", err)
	}

	var output bytes.Buffer
	if err := WriteMagicaVoxel(context.Background(), &output, voxels); err != nil {
		t.Fatalf("WriteMagicaVoxel returned error: %v", err)
	}
	data := output.Bytes()
	if string(data[:4]) != "VOX " || string(data[20:24]) != "SIZE" {
		t.Fatalf("output is not a MagicaVoxel file")
	}
	if got := [3]uint32{
		binary.LittleEndian.Uint32(data[32:36]),
		binary.LittleEndian.Uint32(data[36:40]),
		binary.LittleEndian.Uint32(data[40:44]),
	}; got != [3]uint32{5, 4, 4} {
		t.Fatalf("renderer-axis dimensions = %v, want [5 4 4]", got)
	}
	xyzi_offset := 44
	palette_offset := xyzi_offset + 12 + int(binary.LittleEndian.Uint32(data[xyzi_offset+4:xyzi_offset+8]))
	if string(data[palette_offset:palette_offset+4]) != "RGBA" {
		t.Fatalf("output is missing its embedded palette")
	}
	if got := data[palette_offset+12 : palette_offset+20]; !bytes.Equal(got, []byte{255, 0, 0, 255, 255, 255, 0, 255}) {
		t.Fatalf("front/back palette entries = %v, want red and yellow", got)
	}
}
