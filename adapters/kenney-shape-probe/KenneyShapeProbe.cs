using System.Diagnostics;
using System.Reflection;
using MelonLoader;

[assembly: MelonInfo(typeof(KenneyShapeProbe), "Kenney Shape Pipeline Probe", "0.2.0", "Cataclysm-BN-3D")]
[assembly: MelonGame("Kenney", "Kenney Shape")]

public sealed class KenneyShapeProbe : MelonMod
{
    private static string? pendingExportPath;
    private MethodInfo? add_voxel_method;
    private MethodInfo? get_key_down_method;
    private object? export_key_code;
    private bool installation_validated;
    private bool game_assembly_inspected;
    private bool automation_ready;
    private bool headless_export_started;
    private bool reported_waiting_for_assembly;
    private bool headless_export_requested;

    public override void OnInitializeMelon()
    {
        var process_name = Process.GetCurrentProcess().ProcessName;
        var application_directory = AppDomain.CurrentDomain.BaseDirectory;
        var game_assembly_path = Path.Combine(
                                     application_directory,
                                     "Kenney Shape_Data",
                                     "Managed",
                                     "Assembly-CSharp.dll");
        if (!File.Exists(game_assembly_path))
        {
            LoggerInstance.Error($"Refusing to inspect '{process_name}' from '{application_directory}': Kenney Shape's managed game assembly was not found.");
            return;
        }

        installation_validated = true;
        headless_export_requested = !string.IsNullOrWhiteSpace(
                                       Environment.GetEnvironmentVariable("BN3D_KENNEY_EXPORT_PATH"));
        LoggerInstance.Msg($"Kenney Shape installation detected: {process_name} at {application_directory}");
        LoggerInstance.Msg($"Headless export requested: {headless_export_requested}");
        InspectLoadedGameAssembly();
    }

    public override void OnUpdate()
    {
        if (!installation_validated)
        {
            return;
        }

        if (!automation_ready)
        {
            TryConfigureAutomation();
        }

        if (automation_ready && headless_export_requested && !headless_export_started)
        {
            headless_export_started = true;
            ExportHeadlessProject();
            return;
        }

        if (automation_ready && get_key_down_method is not null &&
            (bool)get_key_down_method.Invoke(null, new[] { export_key_code })!)
        {
            ExportCurrentModel();
        }
    }

    public override void OnSceneWasInitialized(int build_index, string scene_name)
    {
        if (installation_validated && !game_assembly_inspected)
        {
            LoggerInstance.Msg($"Checking game assembly after scene initialization: {scene_name} ({build_index})");
            InspectLoadedGameAssembly();
        }

        if (installation_validated && !automation_ready)
        {
            TryConfigureAutomation();
        }
    }

    private void TryConfigureAutomation()
    {
        var assemblies = AppDomain.CurrentDomain.GetAssemblies();
        var game_assembly = assemblies.FirstOrDefault(assembly => assembly.GetName().Name == "Assembly-CSharp");
        var vox_assembly = assemblies.FirstOrDefault(assembly => assembly.GetType(
                                "Pancake.Tools.World.VoxFile", throwOnError: false) is not null);
        if (game_assembly is null || vox_assembly is null)
        {
            if (!reported_waiting_for_assembly)
            {
                LoggerInstance.Msg("Waiting for Kenney Shape's project and voxel assemblies before enabling export.");
                reported_waiting_for_assembly = true;
            }
            return;
        }

        var export_type = game_assembly.GetType("ExportVoxel", throwOnError: false);
        var add_voxel_method = export_type?.GetMethod(
                                "AddVoxel",
                                BindingFlags.Public | BindingFlags.Static,
                                binder: null,
                                types: new[]
                                {
                                    vox_assembly.GetType("Pancake.Tools.World.VoxFileModel", throwOnError: true)!,
                                    assemblies.Select(assembly => assembly.GetType("UnityEngine.Vector3", throwOnError: false))
                                    .First(type => type is not null)!,
                                    typeof( int ), typeof( int )
                                },
                                modifiers: null);
        if (add_voxel_method is null)
        {
            LoggerInstance.Error("Kenney Shape voxel APIs did not match the inspected signatures; automated export remains disabled.");
            installation_validated = false;
            return;
        }

        var unity_input_type = assemblies
                               .Select(assembly => assembly.GetType("UnityEngine.Input", throwOnError: false))
                               .FirstOrDefault(type => type is not null);
        var key_code_type = assemblies
                            .Select(assembly => assembly.GetType("UnityEngine.KeyCode", throwOnError: false))
                            .FirstOrDefault(type => type is not null);
        get_key_down_method = unity_input_type?.GetMethod(
                                  "GetKeyDown",
                                  BindingFlags.Public | BindingFlags.Static,
                                  binder: null,
                                  types: new[] { key_code_type! },
                                  modifiers: null);
        if (get_key_down_method is not null && key_code_type is not null)
        {
            export_key_code = Enum.Parse(key_code_type, "F8");
        }
        this.add_voxel_method = add_voxel_method;
        automation_ready = true;
        LoggerInstance.Msg(get_key_down_method is null
                               ? "Headless export adapter initialized; no keyboard polling is required."
                               : "F8 export enabled. Press F8 to export the active Kenney Shape voxel model to the configured output folder.");
    }

    private bool ExportCurrentModel(string? requested_path = null)
    {
        var output_directory = Environment.GetEnvironmentVariable("BN3D_VOX_OUTPUT_DIRECTORY");
        if (string.IsNullOrWhiteSpace(requested_path) && string.IsNullOrWhiteSpace(output_directory))
        {
            output_directory = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "Exports");
        }

        try
        {
            if (!string.IsNullOrWhiteSpace(requested_path))
            {
                pendingExportPath = Path.GetFullPath(requested_path);
                Directory.CreateDirectory(Path.GetDirectoryName(pendingExportPath)!);
            }
            else
            {
                Directory.CreateDirectory(output_directory!);
                pendingExportPath = Path.Combine(
                                        Path.GetFullPath(output_directory!),
                                        $"kenney-shape-{DateTime.UtcNow:yyyyMMdd-HHmmss-fff}-{Guid.NewGuid():N}.vox");
            }

            LoggerInstance.Msg($"Writing active voxel model to {pendingExportPath}");
            if (WriteProjectVox(pendingExportPath))
            {
                LoggerInstance.Msg($"Voxel export completed: {pendingExportPath}");
                return true;
            }

            LoggerInstance.Error("Kenney's voxel writer returned without creating the requested VOX file.");
        }
        catch (Exception error)
        {
            LoggerInstance.Error($"Automated voxel export failed: {error}");
        }
        finally
        {
            pendingExportPath = null;
        }

        return false;
    }

    private bool WriteProjectVox(string output_path)
    {
        var assemblies = AppDomain.CurrentDomain.GetAssemblies();
        var game_assembly = assemblies.First(assembly => assembly.GetName().Name == "Assembly-CSharp");
        var vox_assembly = assemblies.First(assembly => assembly.GetType(
                               "Pancake.Tools.World.VoxFile", throwOnError: false) is not null);
        var project_type = game_assembly.GetType("Project", throwOnError: true)!;
        var data = project_type.GetField("data", BindingFlags.Public | BindingFlags.Static)!.GetValue(null)!;
        var data_type = data.GetType();
        var size = data_type.GetField("size")!.GetValue(data)!;
        var size_type = size.GetType();
        var width = (int)size_type.GetProperty("x")!.GetValue(size)!;
        var height = (int)size_type.GetProperty("y")!.GetValue(size)!;
        var depth_multiplier = Convert.ToSingle(data_type.GetField("depthMultiplier")!.GetValue(data));
        var tiles = (Array)data_type.GetField("tiles")!.GetValue(data)!;
        var tile_type = tiles.GetType().GetElementType()!;
        var color_field = tile_type.GetField("color")!;
        var depth_field = tile_type.GetField("depth")!;
        var depth_back_field = tile_type.GetField("depthBack")!;
        var alignment = Convert.ToInt32(data_type.GetField("alignment")!.GetValue(data));
        var initial_depth = alignment == 0 ? height : 0;
        var tile_depths = new TileDepth[tiles.Length];
        var voxel_count = 0;
        var minimum_depth = initial_depth;
        var maximum_depth = initial_depth;
        for (var index = 0; index < tiles.Length; index++)
        {
            var tile = tiles.GetValue(index)!;
            if ((int)color_field.GetValue(tile)! == -1)
            {
                continue;
            }

            var depths = GetScaledDepths(
                             (int)depth_field.GetValue(tile)!,
                             (int)depth_back_field.GetValue(tile)!,
                             depth_multiplier);
            tile_depths[index] = depths;
            voxel_count += depths.front + depths.back - 1;
            minimum_depth = Math.Min(minimum_depth, initial_depth - depths.front + 1);
            maximum_depth = Math.Max(maximum_depth, initial_depth + depths.back - 1);
        }

        var depth_size = maximum_depth - minimum_depth + 1;
        if (width is < 1 or > 256 || height is < 1 or > 256 || depth_size is < 1 or > 256 || voxel_count > 1_000_000)
        {
            throw new InvalidDataException($"Generated voxel model exceeds renderer limits ({width}x{depth_size}x{height}, {voxel_count} voxels).");
        }

        var vox_file_type = vox_assembly.GetType("Pancake.Tools.World.VoxFile", throwOnError: true)!;
        var vox_file = Activator.CreateInstance(vox_file_type, new object[] { width, depth_size, height })!;
        var colors = (System.Collections.IList)data_type.GetField("colors")!.GetValue(data)!;
        var color_array = (Array)vox_file_type.GetMethod("get_Colors")!.Invoke(vox_file, null)!;
        var material_array = (Array)vox_file_type.GetMethod("get_Materials")!.Invoke(vox_file, null)!;
        var color_type = vox_assembly.GetType("Pancake.Tools.World.VoxFileColor", throwOnError: true)!;
        var material_type = vox_assembly.GetType("Pancake.Tools.World.VoxFileMaterial", throwOnError: true)!;
        var color_ctor = color_type.GetConstructor(new[] { typeof( int ), typeof( int ), typeof( int ), typeof( int ) })!;
        var material_ctor = material_type.GetConstructors()
                          .First(ctor => ctor.GetParameters().Length == 13);
        for (var i = 0; i < colors.Count; i++)
        {
            var hex = ((string)colors[i]!).Replace("#", string.Empty);
            var red = int.Parse(hex.Substring(0, 2), System.Globalization.NumberStyles.HexNumber);
            var green = int.Parse(hex.Substring(2, 2), System.Globalization.NumberStyles.HexNumber);
            var blue = int.Parse(hex.Substring(4, 2), System.Globalization.NumberStyles.HexNumber);
            color_array.SetValue(color_ctor.Invoke(new object[] { red, green, blue, 255 }), i);
            material_array.SetValue(material_ctor.Invoke(new object[]
            {
                i, "_diffuse", 1.0f, 0.1f, 0.5f, 0.5f, 0.3f, 0.0f, -0.5f, 0.8f, 0.7f, 0.0f, 0.0f
            }), i);
        }

        var models = (System.Collections.IList)vox_file_type.GetMethod("get_Models")!.Invoke(vox_file, null)!;
        var model = models[0]!;
        var model_type = model.GetType();
        model_type.GetField("SizeY")!.SetValue(model, depth_size);
        model_type.GetMethod("CreateVoxelArray")!.Invoke(model, new object[] { voxel_count });
        var add_voxel = add_voxel_method!;
        var vector3_type = assemblies.Select(assembly => assembly.GetType("UnityEngine.Vector3", throwOnError: false))
                          .First(type => type is not null)!;
        var vector3_ctor = vector3_type.GetConstructor(new[] { typeof( float ), typeof( float ), typeof( float ) })!;
        var voxel_index = 0;
        var adjusted_initial_depth = initial_depth - minimum_depth;
        var color_back_field = tile_type.GetField("colorBack")!;

        for (var x = 0; x < width; x++)
        {
            for (var y = 0; y < height; y++)
            {
                var tile = tiles.GetValue(x * height + y)!;
                var color = (int)color_field.GetValue(tile)!;
                if (color == -1)
                {
                    continue;
                }

                var depths = tile_depths[x * height + y];
                var position = vector3_ctor.Invoke(new object[] { (float)x, (float)y, (float)adjusted_initial_depth });
                add_voxel.Invoke(null, new[] { model, position, color, voxel_index++ });
                for (var depth = 0; depth < depths.front - 1; depth++)
                {
                    position = vector3_ctor.Invoke(new object[] { (float)x, (float)y, (float)(adjusted_initial_depth - 1 - depth) });
                    add_voxel.Invoke(null, new[] { model, position, color, voxel_index++ });
                }

                var back_color = (int)color_back_field.GetValue(tile)!;
                if (back_color == -1) back_color = color;
                for (var depth = 0; depth < depths.back - 1; depth++)
                {
                    position = vector3_ctor.Invoke(new object[] { (float)x, (float)y, (float)(adjusted_initial_depth + 1 + depth) });
                    add_voxel.Invoke(null, new[] { model, position, back_color, voxel_index++ });
                }
            }
        }

        var vox_io_type = vox_assembly.GetType("Pancake.Tools.World.VoxFileIO", throwOnError: true)!;
        var save_file = vox_io_type.GetMethod("SaveFile", BindingFlags.Public | BindingFlags.Static)!;
        save_file.Invoke(null, new[] { output_path, vox_file });
        return File.Exists(output_path);
    }

    private struct TileDepth
    {
        public int front;
        public int back;
    }

    private static TileDepth GetScaledDepths(int front, int back, float multiplier)
    {
        if (front == 0) front = back;
        if (back == 0) back = front;
        if (front == 0 && back == 0) front = back = 1;
        return new TileDepth
        {
            front = (int)Math.Ceiling(front * multiplier / 100.0),
            back = (int)Math.Ceiling(back * multiplier / 100.0)
        };
    }

    private void ExportHeadlessProject()
    {
        var output_path = Environment.GetEnvironmentVariable("BN3D_KENNEY_EXPORT_PATH");
        if (string.IsNullOrWhiteSpace(output_path))
        {
            return;
        }

        var game_assembly = AppDomain.CurrentDomain.GetAssemblies()
                            .First(assembly => assembly.GetName().Name == "Assembly-CSharp");
        var project_type = game_assembly.GetType("Project", throwOnError: true)!;
        var input_path = Environment.GetEnvironmentVariable("BN3D_KENNEY_PROJECT_PATH");
        if (string.IsNullOrWhiteSpace(input_path) || !File.Exists(input_path) ||
            !input_path.EndsWith(".kenshape", StringComparison.OrdinalIgnoreCase))
        {
            LoggerInstance.Error("Headless Kenney export requires BN3D_KENNEY_PROJECT_PATH to name an existing .kenshape project file.");
            RequestApplicationQuit();
            return;
        }

        try
        {
            var load_path_method = project_type.GetMethod(
                                      "LoadPath",
                                      BindingFlags.Public | BindingFlags.Static,
                                      binder: null,
                                      types: new[] { typeof( string ) },
                                      modifiers: null);
            if (load_path_method is null)
            {
                throw new MissingMethodException("Project.LoadPath(string)");
            }

            LoggerInstance.Msg($"Loading Kenney Shape project in batch mode: {input_path}");
            load_path_method.Invoke(null, new object[] { Path.GetFullPath(input_path) });
        }
        catch (Exception error)
        {
            LoggerInstance.Error($"Headless Kenney project loading failed: {error}");
            RequestApplicationQuit();
            return;
        }

        var project_path = (string?)project_type.GetField("path", BindingFlags.Public | BindingFlags.Static)?.GetValue(null);
        if (string.IsNullOrWhiteSpace(project_path) || !File.Exists(project_path) ||
            !project_path.EndsWith(".kenshape", StringComparison.OrdinalIgnoreCase))
        {
            LoggerInstance.Error("Kenney Shape did not load the requested .kenshape project.");
            RequestApplicationQuit();
            return;
        }

        var exported = ExportCurrentModel(output_path);
        if (!exported)
        {
            LoggerInstance.Error("Headless Kenney export failed; verify the project and output path.");
        }

        var manager_type = game_assembly.GetType("ProjectManager", throwOnError: true)!;
        var manager = manager_type.GetProperty("instance", BindingFlags.Public | BindingFlags.Static)!.GetValue(null)!;
        manager_type.GetMethod("SetSaved", BindingFlags.Public | BindingFlags.Instance)!
        .Invoke(manager, new object[] { true });
        RequestApplicationQuit();
    }

    private static void RequestApplicationQuit()
    {
        var application_type = AppDomain.CurrentDomain.GetAssemblies()
                               .Select(assembly => assembly.GetType("UnityEngine.Application", throwOnError: false))
                               .First(type => type is not null)!;
        application_type.GetMethod("Quit", BindingFlags.Public | BindingFlags.Static, null, Type.EmptyTypes, null)!
        .Invoke(null, null);
    }

    private void InspectLoadedGameAssembly()
    {
        var game_assembly = AppDomain.CurrentDomain.GetAssemblies()
                            .FirstOrDefault(assembly => assembly.GetName().Name == "Assembly-CSharp");
        if (game_assembly is null)
        {
            if (!reported_waiting_for_assembly)
            {
                LoggerInstance.Msg("Kenney Shape was detected, but Assembly-CSharp has not loaded yet. The probe will retry after scene initialization.");
                reported_waiting_for_assembly = true;
            }
            return;
        }

        game_assembly_inspected = true;
        LoggerInstance.Msg($"Inspecting game assembly: {game_assembly.Location}");
        foreach (var type in GetLoadableTypes(game_assembly))
        {
            var type_name = type.FullName ?? type.Name;
            var methods = type.GetMethods(BindingFlags.Public | BindingFlags.NonPublic |
                                          BindingFlags.Instance | BindingFlags.Static |
                                          BindingFlags.DeclaredOnly)
                          .Where(method => method.Name.Contains("export", StringComparison.OrdinalIgnoreCase) ||
                                 method.Name.Contains("save", StringComparison.OrdinalIgnoreCase) ||
                                 method.Name.Contains("voxel", StringComparison.OrdinalIgnoreCase))
                          .Select(method => method.ToString())
                          .Distinct(StringComparer.Ordinal)
                          .OrderBy(signature => signature, StringComparer.Ordinal)
                          .ToArray();
            if (methods.Length > 0)
            {
                LoggerInstance.Msg($"Candidate API: {type_name}: {string.Join(" | ", methods)}");
            }
        }
    }

    private static IEnumerable<Type> GetLoadableTypes(Assembly assembly)
    {
        try
        {
            return assembly.GetTypes();
        }
        catch (ReflectionTypeLoadException error)
        {
            return error.Types.OfType<Type>();
        }
    }
}
