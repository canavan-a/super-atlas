package com.superatlasapp

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import com.facebook.react.ReactPackage
import com.facebook.react.bridge.NativeModule
import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod
import com.facebook.react.uimanager.ViewManager
import com.google.firebase.FirebaseApp
import com.google.firebase.messaging.FirebaseMessaging

/**
 * Firebase Cloud Messaging for the Google Play build (src/notifications.ts):
 * hands JS this device's FCM token, which it registers with the server
 * (PUT /push/token). The server sends "notification" messages, which Android
 * shows itself while the app is in the background; PlayMessagingService shows
 * them while it's open.
 */
class PushModule(private val context: ReactApplicationContext) : ReactContextBaseJavaModule(context) {
  override fun getName() = "Push"

  /** False without Google Play services, or without Firebase settings in
   *  res/values/google_services.xml. */
  @ReactMethod
  fun isAvailable(promise: Promise) {
    val hasPlayServices =
        try {
          context.packageManager.getPackageInfo("com.google.android.gms", 0)
          true
        } catch (e: Exception) {
          false
        }
    promise.resolve(hasPlayServices && FirebaseApp.getApps(context).isNotEmpty())
  }

  @ReactMethod
  fun getToken(promise: Promise) {
    ensureChannel(context)
    try {
      FirebaseMessaging.getInstance().token.addOnCompleteListener {
        if (it.isSuccessful) promise.resolve(it.result) else promise.reject("push", it.exception?.message ?: "no token")
      }
    } catch (e: Exception) {
      promise.reject("push", e)
    }
  }

  companion object {
    // Also named in AndroidManifest.xml as FCM's default channel.
    const val CHANNEL = "superatlas-friends"

    fun ensureChannel(context: Context) {
      if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
      val nm = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
      nm.createNotificationChannel(
          NotificationChannel(CHANNEL, "Friends", NotificationManager.IMPORTANCE_DEFAULT).apply {
            description = "Friend requests and new friends."
          })
    }
  }
}

class PushPackage : ReactPackage {
  override fun createNativeModules(context: ReactApplicationContext): List<NativeModule> =
      listOf(PushModule(context))

  override fun createViewManagers(context: ReactApplicationContext): List<ViewManager<*, *>> = emptyList()
}
