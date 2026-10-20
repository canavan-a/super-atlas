package expo.modules.appicon

import android.content.ComponentName
import android.content.Context
import android.content.pm.PackageManager
import expo.modules.kotlin.exception.Exceptions
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition

/**
 * Switches the launcher icon between the per-theme versions, ported from
 * super-chat's AppIconModule.
 *
 * Android has no API to change an app's icon directly. The manifest instead
 * declares one launcher entry per theme (an <activity-alias> named IconLight,
 * IconDark, ..., each with its own icon; see plugins/withAppIcon.js) and
 * exactly one of them is enabled at any time; enabling another swaps the icon
 * the launcher shows. The JS side (src/appIcon.ts) only calls this while the
 * app is going to the background, because some launchers flicker, drop a
 * pinned shortcut, or restart the app when the entry they point at changes.
 */
class AppIconModule : Module() {
  private val context: Context
    get() = appContext.reactContext ?: throw Exceptions.ReactContextLost()

  private val themes = listOf("light", "dark", "slate", "sepia", "ember")

  private fun alias(theme: String) =
      ComponentName(context.packageName, context.packageName + ".Icon" + theme.replaceFirstChar { it.uppercase() })

  /** The manifest enables IconDark by default; the others start disabled. */
  private fun isEnabled(theme: String): Boolean =
      when (context.packageManager.getComponentEnabledSetting(alias(theme))) {
        PackageManager.COMPONENT_ENABLED_STATE_ENABLED -> true
        PackageManager.COMPONENT_ENABLED_STATE_DEFAULT -> theme == DEFAULT
        else -> false
      }

  override fun definition() = ModuleDefinition {
    Name("AppIcon")

    /** The theme whose icon is currently showing. */
    AsyncFunction("getIcon") {
      themes.firstOrNull { isEnabled(it) } ?: DEFAULT
    }

    /** Makes [name]'s icon the launcher icon. */
    AsyncFunction("setIcon") { name: String ->
      require(name in themes) { "Unknown icon: $name" }
      val pm = context.packageManager
      // Enable the new entry before disabling the old ones, so there is never
      // a moment with no launcher entry at all. DONT_KILL_APP keeps the app
      // running while its own launcher entry changes.
      pm.setComponentEnabledSetting(
          alias(name), PackageManager.COMPONENT_ENABLED_STATE_ENABLED, PackageManager.DONT_KILL_APP)
      for (t in themes) {
        if (t != name) {
          pm.setComponentEnabledSetting(
              alias(t), PackageManager.COMPONENT_ENABLED_STATE_DISABLED, PackageManager.DONT_KILL_APP)
        }
      }
      true
    }
  }

  companion object {
    private const val DEFAULT = "dark"
  }
}
