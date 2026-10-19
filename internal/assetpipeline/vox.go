package assetpipeline

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type VOXInfo struct {
	Version      uint32
	Size         [3]uint32
	VoxelCount   uint32
	HasPalette   bool
	PaletteBytes int
}

type voxManifest struct {
	Format                string  `json:"format"`
	Source                string  `json:"source"`
	SHA256                string  `json:"sha256"`
	AssetKind             string  `json:"asset_kind"`
	AssetID               *string `json:"asset_id"`
	VOXVersion            uint32  `json:"vox_version"`
	Dimensions            voxSize `json:"dimensions"`
	VoxelCount            uint32  `json:"voxel_count"`
	HasEmbeddedPalette    bool    `json:"has_embedded_palette"`
	RendererCompatibility string  `json:"renderer_compatibility"`
}

type voxSize struct {
	X uint32 `json:"x"`
	Y uint32 `json:"y"`
	Z uint32 `json:"z"`
}

type voxChunk struct {
	id      string
	content []byte
}

// InspectVOX validates the renderer's single-model MagicaVoxel subset.
func InspectVOX(path string) (VOXInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return VOXInfo{}, fmt.Errorf("read VOX file: %w", err)
	}
	if len(data) < 20 || string(data[:4]) != "VOX " {
		return VOXInfo{}, fmt.Errorf("not a MagicaVoxel VOX file")
	}
	version := binary.LittleEndian.Uint32(data[4:8])
	rootID := string(data[8:12])
	rootContentSize := binary.LittleEndian.Uint32(data[12:16])
	rootChildrenSize := binary.LittleEndian.Uint32(data[16:20])
	if rootID != "MAIN" || rootContentSize != 0 {
		return VOXInfo{}, fmt.Errorf("VOX file must contain an empty MAIN root chunk")
	}
	childrenEnd := uint64(20) + uint64(rootChildrenSize)
	if childrenEnd != uint64(len(data)) {
		return VOXInfo{}, fmt.Errorf("MAIN child size does not match the file length")
	}
	chunks, err := readVOXChunks(data, 20, int(childrenEnd))
	if err != nil {
		return VOXInfo{}, err
	}

	var sizeChunks, voxelChunks, paletteChunks [][]byte
	for _, chunk := range chunks {
		switch chunk.id {
		case "SIZE":
			sizeChunks = append(sizeChunks, chunk.content)
		case "XYZI":
			voxelChunks = append(voxelChunks, chunk.content)
		case "RGBA":
			paletteChunks = append(paletteChunks, chunk.content)
		}
	}
	if len(sizeChunks) != 1 || len(voxelChunks) != 1 {
		return VOXInfo{}, fmt.Errorf("renderer-compatible VOX assets must contain exactly one SIZE and one XYZI chunk")
	}
	if len(paletteChunks) > 1 {
		return VOXInfo{}, fmt.Errorf("VOX file contains multiple RGBA palette chunks")
	}
	if len(sizeChunks[0]) != 12 {
		return VOXInfo{}, fmt.Errorf("SIZE chunk must contain exactly three 32-bit dimensions")
	}
	size := [3]uint32{
		binary.LittleEndian.Uint32(sizeChunks[0][0:4]),
		binary.LittleEndian.Uint32(sizeChunks[0][4:8]),
		binary.LittleEndian.Uint32(sizeChunks[0][8:12]),
	}
	for _, dimension := range size {
		if dimension < 1 || dimension > 256 {
			return VOXInfo{}, fmt.Errorf("model dimensions must be between 1 and 256 voxels")
		}
	}
	voxelData := voxelChunks[0]
	if len(voxelData) < 4 {
		return VOXInfo{}, fmt.Errorf("truncated XYZI chunk")
	}
	voxelCount := binary.LittleEndian.Uint32(voxelData[:4])
	if uint64(len(voxelData)) != uint64(4)+uint64(voxelCount)*4 {
		return VOXInfo{}, fmt.Errorf("XYZI voxel count does not match the chunk length")
	}
	if voxelCount > size[0]*size[1]*size[2] {
		return VOXInfo{}, fmt.Errorf("XYZI voxel count exceeds the SIZE volume")
	}
	occupied := make(map[[3]byte]struct{}, voxelCount)
	for offset := 4; offset < len(voxelData); offset += 4 {
		coordinate := [3]byte{voxelData[offset], voxelData[offset+1], voxelData[offset+2]}
		colorIndex := voxelData[offset+3]
		if uint32(coordinate[0]) >= size[0] || uint32(coordinate[1]) >= size[1] || uint32(coordinate[2]) >= size[2] {
			return VOXInfo{}, fmt.Errorf("XYZI coordinate falls outside the SIZE dimensions")
		}
		if colorIndex == 0 {
			return VOXInfo{}, fmt.Errorf("XYZI color index 0 is reserved for transparency")
		}
		if _, found := occupied[coordinate]; found {
			return VOXInfo{}, fmt.Errorf("duplicate voxel coordinate %v", coordinate)
		}
		occupied[coordinate] = struct{}{}
	}
	if len(paletteChunks) == 1 && len(paletteChunks[0]) != 1024 {
		return VOXInfo{}, fmt.Errorf("RGBA chunk must contain 256 RGBA colors")
	}
	return VOXInfo{
		Version:      version,
		Size:         size,
		VoxelCount:   voxelCount,
		HasPalette:   len(paletteChunks) == 1,
		PaletteBytes: len(paletteChunks),
	}, nil
}

// WriteVOXManifest validates a VOX model and writes its hash/provenance manifest.
func WriteVOXManifest(inputPath string, outputPath string, assetKind string, assetID string) error {
	info, err := InspectVOX(inputPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read VOX file for hash: %w", err)
	}
	hash := sha256.Sum256(data)
	var manifestAssetID *string
	if assetID != "" {
		manifestAssetID = &assetID
	}
	manifest := voxManifest{
		Format:                "bn3d-vox-asset-v1",
		Source:                filepath.Base(inputPath),
		SHA256:                hex.EncodeToString(hash[:]),
		AssetKind:             assetKind,
		AssetID:               manifestAssetID,
		VOXVersion:            info.Version,
		Dimensions:            voxSize{X: info.Size[0], Y: info.Size[1], Z: info.Size[2]},
		VoxelCount:            info.VoxelCount,
		HasEmbeddedPalette:    info.HasPalette,
		RendererCompatibility: "single-model SIZE/XYZI subset",
	}
	return writeJSON(outputPath, manifest)
}

func readVOXChunks(data []byte, start int, end int) ([]voxChunk, error) {
	chunks := make([]voxChunk, 0)
	for offset := start; offset < end; {
		if end-offset < 12 {
			return nil, fmt.Errorf("truncated VOX chunk header")
		}
		id := string(data[offset : offset+4])
		contentSize := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		childrenSize := uint64(binary.LittleEndian.Uint32(data[offset+8 : offset+12]))
		contentStart := uint64(offset + 12)
		contentEnd := contentStart + contentSize
		chunkEnd := contentEnd + childrenSize
		if chunkEnd > uint64(end) {
			return nil, fmt.Errorf("chunk %s exceeds its parent", id)
		}
		chunks = append(chunks, voxChunk{id: id, content: data[int(contentStart):int(contentEnd)]})
		offset = int(chunkEnd)
	}
	if start != end && (len(chunks) == 0 || len(data) < end) {
		return nil, fmt.Errorf("VOX chunk sizes do not align with the parent boundary")
	}
	return chunks, nil
}
