package voxelizer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/internal/voxelizer/backend"
)

const maxVoxelCount = 1_000_000

// Convert creates a single-model MagicaVoxel file from a four-view image sheet.
func Convert(inputPath string, outputPath string, voxelScale float64) (int64, error) {
	if math.IsNaN(voxelScale) || math.IsInf(voxelScale, 0) || voxelScale < 1 || voxelScale > 1024 {
		return 0, fmt.Errorf("voxel scale must be between 1 and 1024")
	}
	if filepath.Ext(outputPath) != ".vox" {
		return 0, fmt.Errorf("output path must use the .vox extension")
	}

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return 0, fmt.Errorf("read input: %w", err)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(inputBytes))
	if err != nil {
		return 0, fmt.Errorf("decode input image: %w", err)
	}
	if config.Width < 4 || config.Width%4 != 0 {
		return 0, fmt.Errorf("quad sheet width must be a positive multiple of 4 (got %d)", config.Width)
	}
	if config.Width > 4096 || config.Height > 4096 {
		return 0, fmt.Errorf("quad sheet dimensions cannot exceed 4096 pixels per axis")
	}
	viewWidth := config.Width / 4
	depth := max(1, int(math.Round(float64(viewWidth)/2.0/voxelScale)))
	estimatedVoxels := int64(math.Ceil(float64(viewWidth)/voxelScale)) *
		int64(math.Ceil(float64(config.Height)/voxelScale)) * int64(2*depth+1)
	if estimatedVoxels > maxVoxelCount {
		return 0, fmt.Errorf("estimated output of %d voxels exceeds the renderer limit of %d; increase voxel scale or reduce the input", estimatedVoxels, maxVoxelCount)
	}

	settings := backend.Settings{
		Layout:        "quad",
		Shape:         "flat",
		VoxelScale:    voxelScale,
		DepthScale:    1,
		FlatDepth:     1,
		CapsulePower:  2,
		BaseThickness: 2.5,
	}
	voxelMap, estimatedCount, err := backend.ConvertTo3D(context.Background(), base64.StdEncoding.EncodeToString(inputBytes), settings)
	if err != nil {
		return 0, fmt.Errorf("generate voxel model: %w", err)
	}

	outputDirectory := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return 0, fmt.Errorf("create output directory: %w", err)
	}
	outputFile, err := os.CreateTemp(outputDirectory, ".sprite-pipeline-*.vox")
	if err != nil {
		return 0, fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := outputFile.Name()
	defer os.Remove(temporaryPath)
	if err := backend.WriteMagicaVoxel(context.Background(), outputFile, voxelMap); err != nil {
		_ = outputFile.Close()
		return 0, fmt.Errorf("write MagicaVoxel file: %w", err)
	}
	if err := outputFile.Close(); err != nil {
		return 0, fmt.Errorf("close output: %w", err)
	}
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("replace output: %w", err)
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return 0, fmt.Errorf("move completed output into place: %w", err)
	}
	return int64(estimatedCount), nil
}
