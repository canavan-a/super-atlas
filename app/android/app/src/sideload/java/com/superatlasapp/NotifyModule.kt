package com.superatlasapp

import com.facebook.react.ReactPackage
import com.facebook.react.bridge.NativeModule
import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod
import com.facebook.react.uimanager.ViewManager

/** JS control of NotifyService (src/notifications.ts). Sideload build only. */
class NotifyModule(private val context: ReactApplicationContext) : ReactContextBaseJavaModule(context) {
  override fun getName() = "Notify"

  @ReactMethod
  fun isEnabled(promise: Promise) {
    promise.resolve(NotifyService.isEnabled(context))
  }

  /** Must be called with the app in the foreground (Android won't start a
   *  foreground service from the background). */
  @ReactMethod
  fun enable(apiUrl: String, token: String, promise: Promise) {
    try {
      NotifyService.enable(context, apiUrl, token)
      promise.resolve(null)
    } catch (e: Exception) {
      promise.reject("notify_start_failed", e)
    }
  }

  @ReactMethod
  fun disable(promise: Promise) {
    NotifyService.disable(context)
    promise.resolve(null)
  }
}

class NotifyPackage : ReactPackage {
  override fun createNativeModules(context: ReactApplicationContext): List<NativeModule> =
      listOf(NotifyModule(context))

  override fun createViewManagers(context: ReactApplicationContext): List<ViewManager<*, *>> = emptyList()
}
