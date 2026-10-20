package com.superatlasapp

import android.app.Activity
import com.facebook.react.ReactPackage

/**
 * What only the sideloaded build (distributed outside Google Play, see
 * release/release.sh) has: the in-app updater and the background
 * notification connection. The play flavor's Flavor.kt has neither.
 */
object Flavor {
  fun packages(): List<ReactPackage> = listOf(AppUpdaterPackage(), NotifyPackage())

  fun onResume(activity: Activity) {}

  fun onPause() {}
}
