using System.Diagnostics;
using System.Reflection;
using MelonLoader;

[assembly: MelonInfo(typeof(KenneyShapeProbe), "Kenney Shape Pipeline Probe", "0.1.0", "Cataclysm-BN-3D")]

public sealed class KenneyShapeProbe : MelonMod
{
    public override void OnInitializeMelon()
    {
        var processName = Process.GetCurrentProcess().ProcessName;
        var applicationDirectory = AppDomain.CurrentDomain.BaseDirectory;
        var assemblyDirectory = Path.Combine(applicationDirectory, "Kenney Shape_Data", "Managed");
        var gameAssemblyPath = Path.Combine(assemblyDirectory, "Assembly-CSharp.dll");
        if (!File.Exists(gameAssemblyPath))
        {
            LoggerInstance.Error($"Refusing to inspect '{processName}' from '{applicationDirectory}': Kenney Shape's managed game assembly was not found.");
            return;
        }

        LoggerInstance.Msg($"Kenney Shape installation detected: {processName} at {applicationDirectory}");
        var assemblies = AppDomain.CurrentDomain.GetAssemblies()
            .OrderBy(assembly => assembly.GetName().Name, StringComparer.Ordinal)
            .ToArray();
        var gameAssemblyLoaded = false;
        foreach (var assembly in assemblies)
        {
            var assemblyName = assembly.GetName().Name ?? "<unnamed>";
            LoggerInstance.Msg($"Loaded assembly: {assemblyName}");
            if (assemblyName != "Assembly-CSharp")
            {
                continue;
            }
            gameAssemblyLoaded = true;
            LoggerInstance.Msg($"Inspecting game assembly: {assembly.Location}");

            foreach (var type in GetLoadableTypes(assembly))
            {
                var typeName = type.FullName ?? type.Name;
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
                    LoggerInstance.Msg($"Candidate API: {typeName}: {string.Join(" | ", methods)}");
                }
            }
        }

        if (!gameAssemblyLoaded)
        {
            LoggerInstance.Msg("Kenney Shape was detected, but Assembly-CSharp has not loaded yet. Review a later log after the main scene starts.");
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
