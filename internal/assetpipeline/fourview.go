package assetpipeline

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
)

var viewOrder = []string{"left", "front", "right", "back"}

type viewManifestEntry struct {
	Direction string `json:"direction"`
	Source    string `json:"source"`
	SHA256    string `json:"sha256"`
	OffsetX   int    `json:"offset_x"`
}

type fourViewManifest struct {
	Format            string              `json:"format"`
	ViewOrder         []string            `json:"view_order"`
	DimensionsPerView imageDimensions     `json:"dimensions_per_view"`
	Output            string              `json:"output"`
	Views             []viewManifestEntry `json:"views"`
	Voxelizer         struct {
		Project string `json:"project"`
		Mode    string `json:"mode"`
	} `json:"voxelizer"`
}

type imageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// PrepareFourView packs four equally sized images in [LEFT, FRONT, RIGHT, BACK] order.
func PrepareFourView(views map[string]string, outputPath string) (string, error) {
	images := make([]*image.RGBA, 0, len(viewOrder))
	entries := make([]viewManifestEntry, 0, len(viewOrder))
	var width int
	var height int
	for _, direction := range viewOrder {
		inputPath := views[direction]
		if inputPath == "" {
			return "", fmt.Errorf("missing %s view", direction)
		}
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return "", fmt.Errorf("read %s view: %w", direction, err)
		}
		decoded, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("decode %s view: %w", direction, err)
		}
		bounds := decoded.Bounds()
		if width == 0 {
			width = bounds.Dx()
			height = bounds.Dy()
		} else if bounds.Dx() != width || bounds.Dy() != height {
			return "", fmt.Errorf("all four views must have identical dimensions (%s=%dx%d, expected %dx%d)", direction, bounds.Dx(), bounds.Dy(), width, height)
		}
		converted := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(converted, converted.Bounds(), decoded, bounds.Min, draw.Src)
		images = append(images, converted)
		hash := sha256.Sum256(data)
		entries = append(entries, viewManifestEntry{
			Direction: direction,
			Source:    filepath.Base(inputPath),
			SHA256:    hex.EncodeToString(hash[:]),
			OffsetX:   (len(entries)) * width,
		})
	}

	if outputPath == "" || filepath.Ext(outputPath) != ".png" {
		return "", fmt.Errorf("output sheet path must use the .png extension")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	sheet := image.NewRGBA(image.Rect(0, 0, width*len(images), height))
	for index, source := range images {
		draw.Draw(sheet, image.Rect(index*width, 0, (index+1)*width, height), source, image.Point{}, draw.Src)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output sheet: %w", err)
	}
	if err := png.Encode(file, sheet); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("encode output sheet: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close output sheet: %w", err)
	}

	manifest := fourViewManifest{
		Format:            "bn3d-four-view-sheet-v1",
		ViewOrder:         viewOrder,
		DimensionsPerView: imageDimensions{Width: width, Height: height},
		Output:            filepath.Base(outputPath),
		Views:             entries,
	}
	manifest.Voxelizer.Project = "https://github.com/GazPrash/2d-to-3d-voxelizer"
	manifest.Voxelizer.Mode = "quad"
	manifestPath := replaceExt(outputPath, ".manifest.json")
	if err := writeJSON(manifestPath, manifest); err != nil {
		return "", err
	}
	return manifestPath, nil
}
