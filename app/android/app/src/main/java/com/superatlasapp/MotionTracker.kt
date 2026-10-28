package com.superatlasapp

import android.Manifest
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.location.Location
import android.os.Build
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.os.SystemClock
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.facebook.react.ReactPackage
import com.facebook.react.bridge.NativeModule
import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod
import com.facebook.react.uimanager.ViewManager
import com.google.android.gms.location.Geofence
import com.google.android.gms.location.GeofencingEvent
import com.google.android.gms.location.GeofencingRequest
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone
import java.util.concurrent.Executors
import org.json.JSONArray
import org.json.JSONObject

/**
 * Motion-triggered location sharing, the way Life360 does it on Android.
 *
 * While you're still, there's one geofence around where you stopped and
 * nothing else running. Walking or driving out of it wakes MotionReceiver
 * (even with the app closed), which starts MotionService: a location
 * foreground service that sends a fix every ~15 s while you move. Once
 * you've stayed within STILL_RADIUS_M for STILL_MS it drops a new geofence
 * where you are and stops itself. A geofence exit is one of the few events
 * Android lets start a location foreground service from the background,
 * which is why this is native: expo-location's task won't start its service
 * unless the app is open.
 *
 * Turned on from src/location/backgroundTask.ts once background location is
 * granted (it hands over the session token; the server URL is baked in), off
 * on sign-out. The OS-scheduled background task there stays as a fallback and
 * re-arms this if the geofence was lost (location switched off, Play
 * services reset); MotionReceiver re-arms it after a reboot or an update.
 * Needs Google Play services for geofencing; without them this does nothing
 * and the fallback is all there is.
 */
object MotionTracker {
  private const val TAG = "MotionTracker"
  const val ACTION_GEOFENCE = "com.superatlasapp.MOTION_GEOFENCE"
  private const val PREFS = "superatlas.motion"
  private const val KEY_TOKEN = "token"
  private const val FENCE_ID = "superatlas-still"
  // Google's advice is 100-150 m at least; smaller fences miss exits.
  private const val FENCE_RADIUS_M = 150f

  private fun prefs(context: Context) = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

  fun token(context: Context): String? = prefs(context).getString(KEY_TOKEN, null)

  fun isEnabled(context: Context) = token(context) != null

  /** Turns it on as this session, and arms the geofence where you are now. */
  fun enable(context: Context, token: String) {
    prefs(context).edit().putString(KEY_TOKEN, token).apply()
    arm(context)
  }

  /** Turns it off and forgets the session. */
  fun disable(context: Context) {
    prefs(context).edit().clear().apply()
    LocationServices.getGeofencingClient(context).removeGeofences(geofenceIntent(context))
    context.stopService(Intent(context, MotionService::class.java))
  }

  /**
   * Puts the geofence around the current position, unless you're already
   * being tracked as moving. Safe to call any time; [done] runs either way.
   */
  @SuppressLint("MissingPermission") // checked by hasBackgroundLocation
  fun arm(context: Context, done: () -> Unit = {}) {
    if (!isEnabled(context) || MotionService.running || !hasBackgroundLocation(context)) {
      done()
      return
    }
    LocationServices.getFusedLocationProviderClient(context)
        .getCurrentLocation(Priority.PRIORITY_BALANCED_POWER_ACCURACY, null)
        .addOnCompleteListener { task ->
          val here = if (task.isSuccessful) task.result else null
          if (here != null) watchFrom(context, here)
          done()
        }
  }

  /** Puts the geofence around [here]: leaving it starts MotionService. */
  @SuppressLint("MissingPermission")
  fun watchFrom(context: Context, here: Location) {
    if (!hasBackgroundLocation(context)) return
    val fence =
        Geofence.Builder()
            .setRequestId(FENCE_ID)
            // A fix that's only good to 300 m needs a fence at least that
            // big, or GPS noise alone would count as leaving.
            .setCircularRegion(here.latitude, here.longitude, maxOf(FENCE_RADIUS_M, here.accuracy))
            .setExpirationDuration(Geofence.NEVER_EXPIRE)
            .setTransitionTypes(Geofence.GEOFENCE_TRANSITION_EXIT)
            .build()
    val request =
        GeofencingRequest.Builder()
            // Fires straight away if we're somehow already outside it.
            .setInitialTrigger(GeofencingRequest.INITIAL_TRIGGER_EXIT)
            .addGeofence(fence)
            .build()
    // Same request ID, so this replaces the previous fence.
    LocationServices.getGeofencingClient(context)
        .addGeofences(request, geofenceIntent(context))
        .addOnFailureListener { Log.w(TAG, "Could not add geofence", it) }
  }

  private fun geofenceIntent(context: Context): PendingIntent =
      PendingIntent.getBroadcast(
          context,
          0,
          Intent(context, MotionReceiver::class.java).setAction(ACTION_GEOFENCE),
          // Play services fills in the event, so it has to be mutable.
          PendingIntent.FLAG_UPDATE_CURRENT or
              (if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) PendingIntent.FLAG_MUTABLE else 0))

  fun hasBackgroundLocation(context: Context): Boolean {
    val permission =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) Manifest.permission.ACCESS_BACKGROUND_LOCATION
        else Manifest.permission.ACCESS_FINE_LOCATION
    return ContextCompat.checkSelfPermission(context, permission) == PackageManager.PERMISSION_GRANTED
  }

  /** Sends one fix to /locations/batch. Returns the HTTP status, or -1 if
   *  it never got an answer. Blocks, so not on the main thread. */
  fun upload(context: Context, fix: Location): Int {
    val token = token(context) ?: return -1
    val iso =
        SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply { timeZone = TimeZone.getTimeZone("UTC") }
    val point =
        JSONObject()
            .put("latitude", fix.latitude)
            .put("longitude", fix.longitude)
            .put("recordedAt", iso.format(fix.time))
    if (fix.hasAccuracy()) point.put("accuracyM", fix.accuracy.toDouble())
    val body = JSONObject().put("points", JSONArray().put(point)).toString().toByteArray()
    return try {
      val conn = URL(BuildConfig.API_URL + "/locations/batch").openConnection() as HttpURLConnection
      try {
        conn.requestMethod = "POST"
        conn.connectTimeout = 15_000
        conn.readTimeout = 15_000
        conn.doOutput = true
        conn.setRequestProperty("Content-Type", "application/json")
        conn.setRequestProperty("Authorization", "Bearer $token")
        conn.outputStream.use { it.write(body) }
        conn.responseCode
      } finally {
        conn.disconnect()
      }
    } catch (e: Exception) {
      -1
    }
  }
}

/** Geofence exits (start tracking), and reboots and app updates (which
 *  clear every geofence, so re-arm). */
class MotionReceiver : BroadcastReceiver() {
  override fun onReceive(context: Context, intent: Intent) {
    when (intent.action) {
      MotionTracker.ACTION_GEOFENCE -> {
        val event = GeofencingEvent.fromIntent(intent) ?: return
        if (event.hasError()) return
        if (event.geofenceTransition == Geofence.GEOFENCE_TRANSITION_EXIT && MotionTracker.isEnabled(context)) {
          MotionService.start(context)
        }
      }
      Intent.ACTION_BOOT_COMPLETED,
      Intent.ACTION_MY_PACKAGE_REPLACED -> {
        val pending = goAsync()
        MotionTracker.arm(context) { pending.finish() }
      }
    }
  }
}

/** Runs while you're moving: sends fixes, and stops once you're still. */
class MotionService : Service() {
  private val handler = Handler(Looper.getMainLooper())
  private val uploads = Executors.newSingleThreadExecutor()
  private val fused by lazy { LocationServices.getFusedLocationProviderClient(this) }
  // Where you've been within STILL_RADIUS_M of since stillSince.
  private var anchor: Location? = null
  private var stillSince = 0L
  private var latest: Location? = null

  override fun onBind(intent: Intent?): IBinder? = null

  override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
    ensureChannel(this)
    val ongoing =
        NotificationCompat.Builder(this, CH_MOTION)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle("Super Atlas")
            .setContentText("Sharing your location while you're moving")
            .setContentIntent(openApp())
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .build()
    try {
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
        startForeground(NID_MOTION, ongoing, ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION)
      } else {
        startForeground(NID_MOTION, ongoing)
      }
    } catch (e: Exception) {
      // Location permission was taken away since the geofence was armed.
      Log.w(TAG, "Could not start tracking", e)
      stopSelf()
      return START_NOT_STICKY
    }
    if (!running) {
      running = true
      stillSince = SystemClock.elapsedRealtime()
      requestUpdates()
      handler.postDelayed(checkStill, CHECK_MS)
    }
    // If Android kills this mid-trip, the fallback background task re-arms
    // the geofence where you are, so the next move starts it again.
    return START_NOT_STICKY
  }

  override fun onDestroy() {
    running = false
    handler.removeCallbacksAndMessages(null)
    fused.removeLocationUpdates(callback)
    uploads.shutdown()
    super.onDestroy()
  }

  @SuppressLint("MissingPermission") // startForeground above fails without it
  private fun requestUpdates() {
    val request =
        LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, INTERVAL_MS)
            .setMinUpdateDistanceMeters(MIN_DISTANCE_M)
            .build()
    try {
      fused.requestLocationUpdates(request, callback, Looper.getMainLooper())
    } catch (e: SecurityException) {
      stopSelf()
    }
  }

  private val callback =
      object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
          val fix = result.lastLocation ?: return
          if (fix.hasAccuracy() && fix.accuracy > MAX_ACCURACY_M) return
          latest = fix
          val a = anchor
          if (a == null || fix.distanceTo(a) > STILL_RADIUS_M) {
            anchor = fix
            stillSince = SystemClock.elapsedRealtime()
          }
          uploads.execute {
            if (MotionTracker.upload(this@MotionService, fix) == 401) {
              // Signed out elsewhere: stop for good until the next sign-in.
              handler.post { MotionTracker.disable(this@MotionService) }
            }
          }
        }
      }

  // Checked on a timer rather than per fix, since a phone that isn't moving
  // gets no fixes at all (MIN_DISTANCE_M).
  private val checkStill =
      object : Runnable {
        override fun run() {
          if (SystemClock.elapsedRealtime() - stillSince < STILL_MS) {
            handler.postDelayed(this, CHECK_MS)
            return
          }
          running = false
          val here = latest
          if (here != null) MotionTracker.watchFrom(this@MotionService, here)
          else MotionTracker.arm(this@MotionService)
          stopSelf()
        }
      }

  private fun openApp(): PendingIntent =
      PendingIntent.getActivity(
          this,
          0,
          Intent(this, MainActivity::class.java)
              .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP),
          PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT)

  companion object {
    private const val TAG = "MotionService"
    private const val CH_MOTION = "superatlas-motion"
    private const val NID_MOTION = 0x5a02
    private const val INTERVAL_MS = 15_000L
    private const val MIN_DISTANCE_M = 20f
    // Worse than this is a cell-tower guess, not a position.
    private const val MAX_ACCURACY_M = 100f
    // Still means staying within this of one spot for this long: longer
    // than a red light, shorter than parking and walking off.
    private const val STILL_RADIUS_M = 75f
    private const val STILL_MS = 5 * 60_000L
    private const val CHECK_MS = 30_000L

    /** True while this process has the service tracking. */
    @Volatile var running = false
      private set

    fun start(context: Context) {
      try {
        ContextCompat.startForegroundService(context, Intent(context, MotionService::class.java))
      } catch (e: Exception) {
        // Android refused a background start (it shouldn't for a geofence).
        Log.w(TAG, "Could not start tracking", e)
      }
    }

    private fun ensureChannel(context: Context) {
      if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
      val nm = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
      nm.createNotificationChannel(
          NotificationChannel(CH_MOTION, "Location while moving", NotificationManager.IMPORTANCE_LOW).apply {
            description = "Shown while Super Atlas shares your location on the move."
            setShowBadge(false)
          })
    }
  }
}

/** JS control of MotionTracker (src/location/backgroundTask.ts). */
class MotionModule(private val context: ReactApplicationContext) : ReactContextBaseJavaModule(context) {
  override fun getName() = "Motion"

  @ReactMethod
  fun enable(token: String, promise: Promise) {
    MotionTracker.enable(context, token)
    promise.resolve(null)
  }

  @ReactMethod
  fun disable(promise: Promise) {
    MotionTracker.disable(context)
    promise.resolve(null)
  }
}

class MotionPackage : ReactPackage {
  override fun createNativeModules(context: ReactApplicationContext): List<NativeModule> =
      listOf(MotionModule(context))

  override fun createViewManagers(context: ReactApplicationContext): List<ViewManager<*, *>> = emptyList()
}
