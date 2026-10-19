package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/tools/voxelizer-backend/backend"
)

func main() {
	input := flag.String("input", "", "quad sprite sheet image (PNG/JPEG) laid out [LEFT, FRONT, RIGHT, BACK]")
	output := flag.String("output", "", "output MagicaVoxel .vox path")
	voxel_scale := flag.Float64("voxel-scale", 1, "voxelizer input downscale factor; range 1..1024")
	flag.Parse()
	if *input == "" || *output == "" {
		flag.Usage()
		os.Exit(2)
	}
	if math.IsNaN(*voxel_scale) || math.IsInf(*voxel_scale, 0) || *voxel_scale < 1 || *voxel_scale > 1024 {
		log.Fatal("-voxel-scale must be between 1 and 1024")
	}
	if filepath.Ext(*output) != ".vox" {
		log.Fatal("-output must use the .vox extension")
	}

	input_bytes, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("read input: %v", err)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(input_bytes))
	if err != nil {
		log.Fatalf("decode input image: %v", err)
	}
	if config.Width < 4 || config.Width%4 != 0 {
		log.Fatalf("quad sheet width must be a positive multiple of 4 (got %d)", config.Width)
	}
	if config.Width > 4096 || config.Height > 4096 {
		log.Fatal("quad sheet dimensions cannot exceed 4096 pixels per axis")
	}
	view_width := config.Width / 4
	depth := max(1, int(math.Round(float64(view_width)/2.0 / *voxel_scale)))
	estimated_voxels := int64(math.Ceil(float64(view_width) / *voxel_scale)) *
		int64(math.Ceil(float64(config.Height) / *voxel_scale)) * int64(2*depth+1)
	if estimated_voxels > max_voxel_count {
		log.Fatalf("conservative output estimate %d exceeds the renderer's %d-voxel limit; increase -voxel-scale or reduce the input", estimated_voxels, max_voxel_count)
	}
	settings := backend.Settings{
		Layout:        "quad",
		Shape:         "flat",
		VoxelScale:    *voxel_scale,
		DepthScale:    1,
		FlatDepth:     1,
		CapsulePower:  2,
		BaseThickness: 2.5,
	}
	voxel_map, estimated_count, err := backend.ConvertTo3D(context.Background(), base64.StdEncoding.EncodeToString(input_bytes), settings)
	if err != nil {
		log.Fatalf("generate voxel model: %v", err)
	}

	output_directory := filepath.Dir(*output)
	if err := os.MkdirAll(output_directory, 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}
	output_file, err := os.CreateTemp(output_directory, ".voxelizer-*.vox")
	if err != nil {
		log.Fatalf("create temporary output: %v", err)
	}
	temporary_path := output_file.Name()
	defer os.Remove(temporary_path)
	if err := backend.WriteMagicaVoxel(context.Background(), output_file, voxel_map); err != nil {
		_ = output_file.Close()
		_ = os.Remove(temporary_path)
		log.Fatalf("write MagicaVoxel file: %v", err)
	}
	if err := output_file.Close(); err != nil {
		_ = os.Remove(temporary_path)
		log.Fatalf("close output: %v", err)
	}
	if err := os.Rename(temporary_path, *output); err != nil {
		_ = os.Remove(temporary_path)
		log.Fatalf("move completed output into place: %v", err)
	}
	info, err := os.Stat(*output)
	if err != nil {
		log.Fatalf("inspect output: %v", err)
	}
	fmt.Fprintln(os.Stderr, "Note: MagicaVoxel stores one color per voxel; the generated VOX keeps front/back voxel colors and drops per-side face colors.")
	fmt.Printf("Wrote %s (%d bytes, estimated %d input voxels)\n", *output, info.Size(), estimated_count)
}

const max_voxel_count = 1_000_000
