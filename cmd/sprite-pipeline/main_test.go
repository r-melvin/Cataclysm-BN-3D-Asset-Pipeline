package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/internal/assetpipeline"
)

func TestVoxelizeCommandRunsTheFullSpritePipeline(t *testing.T) {
	root := t.TempDir()
	views := make(map[string]string, 4)
	for _, direction := range []string{"left", "front", "right", "back"} {
		path := filepath.Join(root, direction+".png")
		writeSprite(t, path, color.RGBA{R: 80, G: 120, B: 220, A: 255})
		views[direction] = path
	}
	output := filepath.Join(root, "part.vox")
	arguments := []string{"voxelize"}
	for _, direction := range []string{"left", "front", "right", "back"} {
		arguments = append(arguments, "--"+direction, views[direction])
	}
	arguments = append(arguments, "--output", output, "--asset-kind", "vehicle-part", "--asset-id", "part")
	var standardOutput bytes.Buffer
	var standardError bytes.Buffer
	if err := run(arguments, &standardOutput, &standardError); err != nil {
		t.Fatalf("run voxelize: %v (%s)", err, standardError.String())
	}
	info, err := assetpipeline.InspectVOX(output)
	if err != nil {
		t.Fatalf("inspect generated VOX: %v", err)
	}
	if info.VoxelCount == 0 {
		t.Fatal("voxelizer returned an empty model")
	}
	if !strings.Contains(standardOutput.String(), "part.manifest.json") {
		t.Fatalf("command did not report its manifest: %s", standardOutput.String())
	}
	if !strings.Contains(standardError.String(), "per-side face colors") {
		t.Fatalf("command did not report the color tradeoff: %s", standardError.String())
	}
}

func TestVoxelizeCommandAcceptsPreparedSheet(t *testing.T) {
	root := t.TempDir()
	views := make(map[string]string, 4)
	for _, direction := range []string{"left", "front", "right", "back"} {
		path := filepath.Join(root, direction+".png")
		writeSprite(t, path, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		views[direction] = path
	}
	sheet := filepath.Join(root, "sheet.png")
	if _, err := assetpipeline.PrepareFourView(views, sheet); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "part.vox")
	var standardOutput bytes.Buffer
	var standardError bytes.Buffer
	err := run([]string{"voxelize", "--sheet", sheet, "--output", output, "--asset-kind", "character"}, &standardOutput, &standardError)
	if err != nil {
		t.Fatalf("run voxelize from sheet: %v (%s)", err, standardError.String())
	}
	if _, err := assetpipeline.InspectVOX(output); err != nil {
		t.Fatalf("inspect generated VOX: %v", err)
	}
}

func TestCLIHelpHasOneApplicationAndCommands(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"--help"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"sprite-pipeline", "voxelize", "kenney-export", "prepare-four-view", "inspect-vox"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("help output does not contain %q", expected)
		}
	}
}

func TestEverySubcommandHasDescriptiveHelp(t *testing.T) {
	commands := []struct {
		name        string
		description string
		options     []string
		details     []string
	}{
		{name: "voxelize", description: "or four equal-sized view images into a .vox model", options: []string{"-sheet", "-left", "-front", "-right", "-back", "-output", "-voxel-scale", "-asset-kind", "-asset-id", "-force"}},
		{name: "kenney-export", description: "the separate Unity application (not Cataclysm: Bright Nights)", options: []string{"-game-dir", "-project", "-output", "-asset-kind", "-asset-id", "-timeout", "-force"}, details: []string{"Kenney Shape install folder or macOS .app bundle", "find its MelonLoader assemblies and launch that app"}},
		{name: "prepare-four-view", description: "horizontal PNG sheet in LEFT, FRONT, RIGHT, BACK order", options: []string{"-left", "-front", "-right", "-back", "-output"}},
		{name: "inspect-vox", description: "Validate a single-model MagicaVoxel .vox file", options: []string{"-input", "-manifest", "-asset-kind", "-asset-id"}},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			var standardOutput bytes.Buffer
			var standardError bytes.Buffer
			if err := run([]string{command.name, "--help"}, &standardOutput, &standardError); err != nil {
				t.Fatalf("show help: %v", err)
			}
			help := standardOutput.String() + standardError.String()
			if !strings.Contains(help, command.description) {
				t.Errorf("help is missing command description %q: %s", command.description, help)
			}
			for _, detail := range command.details {
				if !strings.Contains(help, detail) {
					t.Errorf("help is missing detail %q: %s", detail, help)
				}
			}
			for _, option := range command.options {
				if !strings.Contains(help, option) {
					t.Errorf("help is missing %s: %s", option, help)
				}
			}
		})
	}
}

func TestREADMEExamplesUseSupportedCommandsAndOptions(t *testing.T) {
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	readmePath := filepath.Join(filepath.Dir(sourcePath), "..", "..", "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	readme = bytes.ReplaceAll(readme, []byte("\r\n"), []byte("\n"))
	examples := regexp.MustCompile(`(?m)^sprite-pipeline (voxelize|kenney-export|prepare-four-view|inspect-vox)([^\r\n]*)$`).FindAllSubmatch(readme, -1)
	if len(examples) < 5 {
		t.Fatalf("expected at least five documented command examples, found %d", len(examples))
	}
	seenCommands := make(map[string]bool)
	for _, example := range examples {
		commandName := string(example[1])
		seenCommands[commandName] = true
		var standardOutput bytes.Buffer
		var standardError bytes.Buffer
		if err := run([]string{commandName, "--help"}, &standardOutput, &standardError); err != nil {
			t.Fatalf("show %s help: %v", commandName, err)
		}
		help := standardOutput.String() + standardError.String()
		for _, option := range regexp.MustCompile(`--[a-z][a-z-]*`).FindAll(example[2], -1) {
			if string(option) == "--help" {
				continue
			}
			if !strings.Contains(help, "-"+strings.TrimPrefix(string(option), "--")) {
				t.Errorf("README uses unsupported option %s for %s", option, commandName)
			}
		}
		arguments, err := splitDocumentedArguments(string(example[0]))
		if err != nil {
			t.Fatalf("split README command %q: %v", example[0], err)
		}
		arguments = append(arguments[1:], "--help")
		if err := run(arguments, &standardOutput, &standardError); err != nil {
			t.Errorf("README command %s arguments are rejected: %v", commandName, err)
		}
	}
	for _, commandName := range []string{"voxelize", "kenney-export", "prepare-four-view", "inspect-vox"} {
		if !seenCommands[commandName] {
			t.Errorf("README has no copy-paste example for %s", commandName)
		}
	}
}

func splitDocumentedArguments(command string) ([]string, error) {
	var arguments []string
	var current strings.Builder
	inQuotes := false
	for _, character := range command {
		switch {
		case character == '"':
			inQuotes = !inQuotes
		case (character == ' ' || character == '\t') && !inQuotes:
			if current.Len() > 0 {
				arguments = append(arguments, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(character)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("unclosed double quote")
	}
	if current.Len() > 0 {
		arguments = append(arguments, current.String())
	}
	return arguments, nil
}

func writeSprite(t *testing.T, path string, fill color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHelpCommandShowsCommandOptions(t *testing.T) {
	var standardOutput bytes.Buffer
	var standardError bytes.Buffer
	if err := run([]string{"help", "voxelize"}, &standardOutput, &standardError); err != nil {
		t.Fatalf("help voxelize: %v", err)
	}
	if !strings.Contains(standardError.String(), "-voxel-scale") {
		t.Errorf("help voxelize did not print voxelize options: %q", standardError.String())
	}
}
