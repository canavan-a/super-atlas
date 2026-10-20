package com.superatlasapp

import com.facebook.react.ReactPackage
import com.facebook.react.bridge.NativeModule
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.uimanager.ViewManager

/**
 * Tells JS which build this is (src/distribution.ts): "sideload" (in-app
 * updater, background notification connection) or "play" (Google Play:
 * Firebase notifications instead), and which server it was built for
 * (-PapiUrl, app/build.gradle). Ported from super-chat.
 */
class DistributionModule(context: ReactApplicationContext) : ReactContextBaseJavaModule(context) {
  override fun getName() = "Distribution"

  override fun getConstants(): Map<String, Any> =
      mapOf("flavor" to BuildConfig.FLAVOR, "apiUrl" to BuildConfig.API_URL)
}

class DistributionPackage : ReactPackage {
  override fun createNativeModules(context: ReactApplicationContext): List<NativeModule> =
      listOf(DistributionModule(context))

  override fun createViewManagers(context: ReactApplicationContext): List<ViewManager<*, *>> = emptyList()
}
