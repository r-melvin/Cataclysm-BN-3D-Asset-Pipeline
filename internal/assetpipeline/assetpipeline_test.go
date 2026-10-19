package assetpipeline

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareFourViewPacksOrderAndManifest(t *testing.T) {
	root := t.TempDir()
	views := map[string]string{}
	viewColors := map[string]color.RGBA{
		"left":  {R: 10, A: 255},
		"front": {G: 20, A: 255},
		"right": {B: 30, A: 255},
		"back":  {R: 40, G: 40, A: 255},
	}
	for direction, fill := range viewColors {
		path := filepath.Join(root, direction+".png")
		writeTestPNG(t, path, 2, 3, fill)
		views[direction] = path
	}
	output := filepath.Join(root, "sheet.png")
	manifestPath, err := PrepareFourView(views, output)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(file)
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 8 || decoded.Bounds().Dy() != 3 {
		t.Fatalf("packed dimensions = %v, want 8x3", decoded.Bounds())
	}
	for index, direction := range viewOrder {
		got := color.RGBAModel.Convert(decoded.At(index*2, 0)).(color.RGBA)
		if got != viewColors[direction] {
			t.Errorf("view %s pixel = %#v, want %#v", direction, got, viewColors[direction])
		}
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest fourViewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Views) != 4 || manifest.Views[3].Direction != "back" || manifest.Views[3].OffsetX != 6 {
		t.Fatalf("manifest view order/offsets are incorrect: %+v", manifest.Views)
	}
}

func TestPrepareFourViewRejectsDimensionMismatch(t *testing.T) {
	root := t.TempDir()
	views := map[string]string{}
	for index, direction := range viewOrder {
		path := filepath.Join(root, direction+".png")
		width := 2
		if index == 2 {
			width = 3
		}
		writeTestPNG(t, path, width, 2, color.RGBA{R: 255, A: 255})
		views[direction] = path
	}
	if _, err := PrepareFourView(views, filepath.Join(root, "out.png")); err == nil {
		t.Fatal("expected unequal dimensions to fail")
	}
}

func TestWriteVOXManifestAndValidation(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "asset.vox")
	if err := os.WriteFile(input, makeTestVOX([3]uint32{2, 2, 2}, [][4]byte{{0, 1, 1, 7}}, false, true), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := InspectVOX(input)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != [3]uint32{2, 2, 2} || info.VoxelCount != 1 || !info.HasPalette {
		t.Fatalf("unexpected VOX info: %+v", info)
	}
	manifestPath := filepath.Join(root, "asset.manifest.json")
	if err := WriteVOXManifest(input, manifestPath, "vehicle-part", "wheel"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["asset_id"] != "wheel" || manifest["voxel_count"] != float64(1) {
		t.Fatalf("unexpected manifest: %s", data)
	}
}

func TestInspectVOXRejectsUnsupportedModelsAndInvalidVoxels(t *testing.T) {
	cases := []struct {
		name   string
		voxels [][4]byte
		models bool
	}{
		{name: "outside", voxels: [][4]byte{{2, 0, 0, 1}}},
		{name: "transparent", voxels: [][4]byte{{0, 0, 0, 0}}},
		{name: "duplicate", voxels: [][4]byte{{0, 0, 0, 1}, {0, 0, 0, 2}}},
		{name: "multiple models", voxels: [][4]byte{{0, 0, 0, 1}}, models: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.vox")
			if err := os.WriteFile(path, makeTestVOX([3]uint32{2, 2, 2}, testCase.voxels, testCase.models, false), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectVOX(path); err == nil {
				t.Fatal("expected unsupported VOX data to fail validation")
			}
		})
	}
}

func writeTestPNG(t *testing.T, path string, width int, height int, fill color.RGBA) {
	t.Helper()
	imageData := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			imageData.SetRGBA(x, y, fill)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, imageData); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func makeTestVOX(size [3]uint32, voxels [][4]byte, addSecondModel bool, palette bool) []byte {
	children := []byte{}
	modelCount := 1
	if addSecondModel {
		modelCount = 2
	}
	for range modelCount {
		sizeContent := make([]byte, 12)
		for index, dimension := range size {
			binary.LittleEndian.PutUint32(sizeContent[index*4:index*4+4], dimension)
		}
		children = append(children, testVOXChunk("SIZE", sizeContent, nil)...)
		voxelContent := make([]byte, 4+len(voxels)*4)
		binary.LittleEndian.PutUint32(voxelContent[:4], uint32(len(voxels)))
		for index, voxel := range voxels {
			copy(voxelContent[4+index*4:4+index*4+4], voxel[:])
		}
		children = append(children, testVOXChunk("XYZI", voxelContent, nil)...)
	}
	if palette {
		children = append(children, testVOXChunk("RGBA", make([]byte, 1024), nil)...)
	}
	root := testVOXChunk("MAIN", nil, children)
	data := []byte{'V', 'O', 'X', ' '}
	data = append(data, 150, 0, 0, 0)
	data = append(data, root...)
	return data
}

func testVOXChunk(id string, content []byte, children []byte) []byte {
	chunk := bytes.NewBuffer(nil)
	_, _ = chunk.WriteString(id)
	_ = binary.Write(chunk, binary.LittleEndian, uint32(len(content)))
	_ = binary.Write(chunk, binary.LittleEndian, uint32(len(children)))
	_, _ = chunk.Write(content)
	_, _ = chunk.Write(children)
	return chunk.Bytes()
}
