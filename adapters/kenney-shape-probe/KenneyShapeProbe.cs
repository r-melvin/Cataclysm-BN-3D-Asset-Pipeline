using System.Diagnostics;
using System.Reflection;
using HarmonyLib;
using MelonLoader;

[assembly: MelonInfo(typeof(KenneyShapeProbe), "Kenney Shape Pipeline Probe", "0.2.0", "Cataclysm-BN-3D")]
[assembly: MelonGame("Kenney", "Kenney Shape")]

public sealed class KenneyShapeProbe : MelonMod
{
    private static string? pendingExportPath;
    private MethodInfo? export_vox_method;
    private MethodInfo? get_key_down_method;
    private object? export_key_code;
    private bool installation_validated;
    private bool game_assembly_inspected;
    private bool automation_ready;
    private bool reported_waiting_for_assembly;
    private HarmonyLib.Harmony? harmony;

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
        LoggerInstance.Msg($"Kenney Shape installation detected: {process_name} at {application_directory}");
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

        if (automation_ready && (bool)get_key_down_method!.Invoke(null, new[] { export_key_code })!)
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
    }

    private void TryConfigureAutomation()
    {
        var assemblies = AppDomain.CurrentDomain.GetAssemblies();
        var game_assembly = assemblies.FirstOrDefault(assembly => assembly.GetName().Name == "Assembly-CSharp");
        var browser_assembly = assemblies.FirstOrDefault(assembly => assembly.GetName().Name == "Assembly-CSharp-firstpass");
        if (game_assembly is null || browser_assembly is null)
        {
            if (!reported_waiting_for_assembly)
            {
                LoggerInstance.Msg("Waiting for Kenney Shape's game and file-browser assemblies before enabling F8 export.");
                reported_waiting_for_assembly = true;
            }
            return;
        }

        var export_type = game_assembly.GetType("ExportVoxel", throwOnError: false);
        var export_method = export_type?.GetMethod(
                                "Export",
                                BindingFlags.Public | BindingFlags.Static,
                                binder: null,
                                types: Type.EmptyTypes,
                                modifiers: null);
        var browser_type = browser_assembly.GetType("Crosstales.FB.FileBrowser", throwOnError: false);
        var filter_type = browser_assembly.GetType("Crosstales.FB.ExtensionFilter", throwOnError: false);
        var save_method = browser_type?.GetMethod(
                              "SaveFile",
                              BindingFlags.Public | BindingFlags.Instance,
                              binder: null,
                              types: new[]
                              {
                                  typeof( string ), typeof( string ), typeof( string ), filter_type?.MakeArrayType()
                              },
                              modifiers: null);
        if (export_method is null || save_method is null)
        {
            LoggerInstance.Error("Kenney Shape export APIs did not match the inspected signatures; automated export remains disabled.");
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
        if (get_key_down_method is null || key_code_type is null)
        {
            LoggerInstance.Error("Unity's legacy keyboard input API is unavailable; automated export remains disabled.");
            installation_validated = false;
            return;
        }

        export_key_code = Enum.Parse(key_code_type, "F8");
        harmony = new HarmonyLib.Harmony("org.cataclysm-bn-3d.kenney-shape-export");
        var prefix = typeof(KenneyShapeProbe).GetMethod(
                         nameof(SaveFilePrefix),
                         BindingFlags.NonPublic | BindingFlags.Static);
        harmony.Patch(save_method, prefix: new HarmonyMethod(prefix));
        export_vox_method = export_method;
        automation_ready = true;
        LoggerInstance.Msg("F8 export enabled. Press F8 to export the active Kenney Shape voxel model to the configured output folder.");
    }

    private void ExportCurrentModel()
    {
        var output_directory = Environment.GetEnvironmentVariable("BN3D_VOX_OUTPUT_DIRECTORY");
        if (string.IsNullOrWhiteSpace(output_directory))
        {
            output_directory = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "Exports");
        }

        try
        {
            Directory.CreateDirectory(output_directory);
            pendingExportPath = Path.Combine(
                                    Path.GetFullPath(output_directory),
                                    $"kenney-shape-{DateTime.UtcNow:yyyyMMdd-HHmmss-fff}-{Guid.NewGuid():N}.vox");
            LoggerInstance.Msg($"Exporting active voxel model to {pendingExportPath}");
            export_vox_method!.Invoke(null, null);
            if (File.Exists(pendingExportPath))
            {
                LoggerInstance.Msg($"Voxel export completed: {pendingExportPath}");
            }
            else
            {
                LoggerInstance.Error("ExportVoxel.Export returned without creating the requested VOX file.");
            }
        }
        catch (Exception error)
        {
            LoggerInstance.Error($"Automated voxel export failed: {error}");
        }
        finally
        {
            pendingExportPath = null;
        }
    }

    private static bool SaveFilePrefix( ref string __result )
    {
        if (string.IsNullOrEmpty(pendingExportPath))
        {
            return true;
        }

        __result = pendingExportPath;
        return false;
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
