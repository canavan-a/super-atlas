package com.superatlasapp

import android.app.Activity
import com.facebook.react.ReactPackage

/**
 * The Google Play build: Play installs updates itself (an app may not), so
 * there's no in-app updater, and notifications come through Firebase Cloud
 * Messaging (PushModule.kt, PlayMessagingService.kt) rather than an always-on
 * connection. See the sideload flavor's Flavor.kt.
 */
object Flavor {
  fun packages(): List<ReactPackage> = listOf(PushPackage())

  fun onResume(activity: Activity) {}

  fun onPause() {}
}
