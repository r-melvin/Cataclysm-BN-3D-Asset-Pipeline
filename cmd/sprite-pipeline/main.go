package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/internal/assetpipeline"
	"github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/internal/kenney"
	"github.com/r-melvin/Cataclysm-BN-3D-Asset-Pipeline/internal/voxelizer"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "sprite-pipeline: error: %v\n", err)
		os.Exit(2)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printUsage(stdout)
		return nil
	}
	switch args[0] {
	case "prepare-four-view":
		return prepareFourViewCommand(args[1:], stdout, stderr)
	case "inspect-vox":
		return inspectVOXCommand(args[1:], stdout, stderr)
	case "voxelize":
		return voxelizeCommand(args[1:], stdout, stderr)
	case "kenney-export":
		return kenneyExportCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, `sprite-pipeline — voxel asset preparation for Cataclysm: Bright Nights

Usage:
  sprite-pipeline <command> [options]

Commands:
  voxelize           Pack four views or convert a prepared quad sheet to VOX
  kenney-export      Headlessly load a Kenney Shape project and export VOX
  prepare-four-view  Pack four equal-sized directional images into a sheet
  inspect-vox        Validate one VOX model and write its asset manifest

Run 'sprite-pipeline <command> --help' for command options.`)
}

func prepareFourViewCommand(args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("prepare-four-view", flag.ContinueOnError)
	flags.SetOutput(stderr)
	setUsage(flags, "Pack four equal-sized images into a horizontal PNG sheet in LEFT, FRONT, RIGHT, BACK order.")
	views := viewFlags(flags)
	output := flags.String("output", "", "output PNG sheet path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireNoArguments(flags); err != nil {
		return err
	}
	if *output == "" {
		return errors.New("--output is required")
	}
	manifest, err := assetpipeline.PrepareFourView(viewValues(views), *output)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Wrote %s and %s\n", *output, manifest)
	return nil
}

func inspectVOXCommand(args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("inspect-vox", flag.ContinueOnError)
	flags.SetOutput(stderr)
	setUsage(flags, "Validate a single-model MagicaVoxel .vox file and write a provenance manifest.")
	input := flags.String("input", "", "input MagicaVoxel .vox path")
	manifest := flags.String("manifest", "", "output manifest path (defaults beside input)")
	assetKind := flags.String("asset-kind", "", "asset kind: vehicle-part or character")
	assetID := flags.String("asset-id", "", "native asset identifier")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireNoArguments(flags); err != nil {
		return err
	}
	if *input == "" {
		return errors.New("--input is required")
	}
	if err := validateAssetKind(*assetKind); err != nil {
		return err
	}
	manifestPath := *manifest
	if manifestPath == "" {
		manifestPath = replaceExtension(*input, ".manifest.json")
	}
	if err := assetpipeline.WriteVOXManifest(*input, manifestPath, *assetKind, *assetID); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Validated %s; wrote %s\n", *input, manifestPath)
	return nil
}

func voxelizeCommand(args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("voxelize", flag.ContinueOnError)
	flags.SetOutput(stderr)
	setUsage(flags, "Convert either one prepared horizontal [LEFT, FRONT, RIGHT, BACK] sheet or four equal-sized view images into a .vox model.")
	sheet := flags.String("sheet", "", "prepared [LEFT, FRONT, RIGHT, BACK] PNG/JPEG sheet")
	views := viewFlags(flags)
	output := flags.String("output", "", "output MagicaVoxel .vox path")
	voxelScale := flags.Float64("voxel-scale", 1, "voxelizer downscale factor (1..1024)")
	assetKind := flags.String("asset-kind", "", "asset kind: vehicle-part or character")
	assetID := flags.String("asset-id", "", "native asset identifier")
	force := flags.Bool("force", false, "replace an existing output and manifest")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireNoArguments(flags); err != nil {
		return err
	}
	if *output == "" {
		return errors.New("--output is required")
	}
	if !strings.EqualFold(filepath.Ext(*output), ".vox") {
		return errors.New("--output must use the .vox extension")
	}
	if err := validateAssetKind(*assetKind); err != nil {
		return err
	}
	manifestPath := replaceExtension(*output, ".manifest.json")
	if !*force && (fileExists(*output) || fileExists(manifestPath)) {
		return errors.New("output or manifest already exists; pass --force to replace it")
	}
	if *voxelScale < 1 || *voxelScale > 1024 {
		return errors.New("--voxel-scale must be between 1 and 1024")
	}
	inputPath, cleanup, err := createVoxelizerInput(*sheet, views)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	estimatedCount, err := voxelizer.Convert(inputPath, *output, *voxelScale)
	if err != nil {
		return err
	}
	if err := assetpipeline.WriteVOXManifest(*output, manifestPath, *assetKind, *assetID); err != nil {
		return err
	}
	fmt.Fprintln(stderr, "Note: MagicaVoxel stores one color per voxel; the output keeps front/back colors and drops per-side face colors.")
	fmt.Fprintf(stdout, "Wrote and validated %s (%d estimated input voxels); manifest: %s\n", *output, estimatedCount, manifestPath)
	return nil
}

func kenneyExportCommand(args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("kenney-export", flag.ContinueOnError)
	flags.SetOutput(stderr)
	setUsage(flags, "Load an existing .kenshape project in Kenney Shape headlessly and export its voxel model. Requires Kenney Shape, MelonLoader, and the .NET 8 SDK.")
	gameDirectory := flags.String("game-dir", "", "installed Kenney Shape directory or app bundle")
	project := flags.String("project", "", "input .kenshape project path")
	output := flags.String("output", "", "output MagicaVoxel .vox path")
	assetKind := flags.String("asset-kind", "", "asset kind: vehicle-part or character")
	assetID := flags.String("asset-id", "", "native asset identifier")
	timeout := flags.Duration("timeout", 3*time.Minute, "maximum application runtime")
	force := flags.Bool("force", false, "replace an existing output and manifest")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireNoArguments(flags); err != nil {
		return err
	}
	if *gameDirectory == "" || *project == "" || *output == "" {
		return errors.New("--game-dir, --project, and --output are required")
	}
	if err := validateAssetKind(*assetKind); err != nil {
		return err
	}
	if *timeout <= 0 {
		return errors.New("--timeout must be greater than zero")
	}
	manifestPath := replaceExtension(*output, ".manifest.json")
	if !*force && (fileExists(*output) || fileExists(manifestPath)) {
		return errors.New("output or manifest already exists; pass --force to replace it")
	}
	if *force {
		if err := removeIfExists(*output); err != nil {
			return fmt.Errorf("remove previous output: %w", err)
		}
		if err := removeIfExists(manifestPath); err != nil {
			return fmt.Errorf("remove previous manifest: %w", err)
		}
	}
	err := kenney.Export(kenney.ExportOptions{
		GameDirectory: *gameDirectory,
		ProjectPath:   *project,
		OutputPath:    *output,
		Timeout:       *timeout,
	})
	if err != nil {
		return err
	}
	if err := assetpipeline.WriteVOXManifest(*output, manifestPath, *assetKind, *assetID); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Headlessly exported and validated %s; manifest: %s\n", *output, manifestPath)
	return nil
}

func createVoxelizerInput(sheetPath string, views map[string]*string) (string, func(), error) {
	providedViews := 0
	for _, name := range []string{"left", "front", "right", "back"} {
		if *views[name] != "" {
			providedViews++
		}
	}
	if sheetPath != "" {
		if providedViews != 0 {
			return "", nil, errors.New("use either --sheet or all four directional views")
		}
		if !fileExists(sheetPath) {
			return "", nil, fmt.Errorf("sheet does not exist: %s", sheetPath)
		}
		return sheetPath, nil, nil
	}
	if providedViews != 4 {
		return "", nil, errors.New("provide --sheet or all four directional views")
	}
	temporaryDirectory, err := os.MkdirTemp("", "bn3d-sprite-pipeline-")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary sheet directory: %w", err)
	}
	temporaryPath := filepath.Join(temporaryDirectory, "views.png")
	if _, err := assetpipeline.PrepareFourView(viewValues(views), temporaryPath); err != nil {
		_ = os.RemoveAll(temporaryDirectory)
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(temporaryDirectory) }
	return temporaryPath, cleanup, nil
}

func viewFlags(flags *flag.FlagSet) map[string]*string {
	views := make(map[string]*string, 4)
	for _, name := range []string{"left", "front", "right", "back"} {
		path := flags.String(name, "", fmt.Sprintf("%s view image path", name))
		views[name] = path
	}
	return views
}

func setUsage(flags *flag.FlagSet, description string) {
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options]\n\n%s\n\nOptions:\n", flags.Name(), description)
		flags.PrintDefaults()
	}
}

func viewValues(flags map[string]*string) map[string]string {
	views := make(map[string]string, 4)
	for name, path := range flags {
		views[name] = *path
	}
	return views
}

func validateAssetKind(value string) error {
	if value != "vehicle-part" && value != "character" {
		return errors.New("--asset-kind must be vehicle-part or character")
	}
	return nil
}

func requireNoArguments(flags *flag.FlagSet) error {
	if len(flags.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func replaceExtension(path string, extension string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + extension
}
