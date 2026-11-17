using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;
public static class BuildPrototype {
 public static void BuildAndroid(){
  EditorSceneManager.NewScene(NewSceneSetup.EmptyScene,NewSceneMode.Single);
  EditorSceneManager.SaveScene(EditorSceneManager.GetActiveScene(),"Assets/Prototype.unity");
  PlayerSettings.companyName="EBash";PlayerSettings.productName="EBash Unity Prototype";
  PlayerSettings.SetApplicationIdentifier(BuildTargetGroup.Android,"org.ebash.unityprototype");
  PlayerSettings.defaultInterfaceOrientation=UIOrientation.LandscapeLeft;
  PlayerSettings.Android.minSdkVersion=AndroidSdkVersions.AndroidApiLevel26;
  PlayerSettings.SetScriptingBackend(BuildTargetGroup.Android,ScriptingImplementation.IL2CPP);
  PlayerSettings.Android.targetArchitectures=AndroidArchitecture.ARM64;
  EditorUserBuildSettings.buildAppBundle=false;
  var path=System.Environment.GetEnvironmentVariable("EBASH_APK_PATH")??"Build/ebash-unity.apk";
  System.IO.Directory.CreateDirectory(System.IO.Path.GetDirectoryName(path));
  var report=BuildPipeline.BuildPlayer(new[]{"Assets/Prototype.unity"},path,BuildTarget.Android,BuildOptions.Development);
  if(report.summary.result!=UnityEditor.Build.Reporting.BuildResult.Succeeded)throw new System.Exception("Android build failed");
 }
}
