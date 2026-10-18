using System.Diagnostics;
using System.Reflection;
using MelonLoader;

[assembly: MelonInfo(typeof(KenneyShapeProbe), "Kenney Shape Pipeline Probe", "0.1.0", "Cataclysm-BN-3D")]

public sealed class KenneyShapeProbe : MelonMod
{
    public override void OnInitializeMelon()
    {
        var processName = Process.GetCurrentProcess().ProcessName;
        if (!processName.Contains("Kenney Shape", StringComparison.OrdinalIgnoreCase))
        {
            LoggerInstance.Error($"Refusing to inspect unexpected process '{processName}'. Run this probe only in Kenney Shape.");
            return;
        }

        LoggerInstance.Msg($"Kenney Shape process detected: {processName}");
        var assemblies = AppDomain.CurrentDomain.GetAssemblies()
            .OrderBy(assembly => assembly.GetName().Name, StringComparer.Ordinal)
            .ToArray();
        foreach (var assembly in assemblies)
        {
            var assemblyName = assembly.GetName().Name ?? "<unnamed>";
            LoggerInstance.Msg($"Loaded assembly: {assemblyName}");
            if (assemblyName != "Assembly-CSharp")
            {
                continue;
            }

            foreach (var type in GetLoadableTypes(assembly))
            {
                var typeName = type.FullName ?? type.Name;
                var methods = type.GetMethods(BindingFlags.Public | BindingFlags.NonPublic |
                                              BindingFlags.Instance | BindingFlags.Static |
                                              BindingFlags.DeclaredOnly)
                              .Where(method => method.Name.Contains("export", StringComparison.OrdinalIgnoreCase) ||
                                     method.Name.Contains("save", StringComparison.OrdinalIgnoreCase) ||
                                     method.Name.Contains("voxel", StringComparison.OrdinalIgnoreCase))
                              .Select(method => method.Name)
                              .Distinct(StringComparer.Ordinal)
                              .OrderBy(name => name, StringComparer.Ordinal)
                              .ToArray();
                if (methods.Length > 0)
                {
                    LoggerInstance.Msg($"Candidate API: {typeName}: {string.Join(", ", methods)}");
                }
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
