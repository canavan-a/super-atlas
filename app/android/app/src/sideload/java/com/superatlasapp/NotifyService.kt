package com.superatlasapp

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import androidx.core.app.NotificationCompat
import java.util.concurrent.TimeUnit
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject

/**
 * The sideloaded build's background notifications: a foreground service that
 * keeps one WebSocket open to the server's /notifications/stream and shows
 * each event ("<name> wants to be friends", "<name> is now your friend") as a
 * system notification. It's native rather than JS so it doesn't depend on a
 * running React instance, and it carries notifications only — never
 * locations. The Google Play build gets the same notifications through FCM
 * instead (play/PushModule.kt).
 *
 * Opt-in from Settings → Background notifications (src/notifications.ts),
 * which hands over the server URL and session token. It survives reboots
 * (BootReceiver) and stops itself if the server says the session is gone.
 */
class NotifyService : Service() {
  private val client =
      OkHttpClient.Builder()
          // Detects a dead connection (say, after a network switch) so it
          // can reconnect instead of silently missing events.
          .pingInterval(30, TimeUnit.SECONDS)
          .build()
  private val handler = Handler(Looper.getMainLooper())
  private var socket: WebSocket? = null
  private var retryMs = MIN_RETRY_MS
  private var stopped = false

  override fun onBind(intent: Intent?): IBinder? = null

  override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
    ensureChannels(this)
    val ongoing =
        NotificationCompat.Builder(this, CH_SERVICE)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle("Super Atlas")
            .setContentText("Connected for friend requests")
            .setContentIntent(openApp(this))
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_MIN)
            .build()
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      startForeground(NID_SERVICE, ongoing, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC)
    } else {
      startForeground(NID_SERVICE, ongoing)
    }
    stopped = false
    if (socket == null) connect()
    return START_STICKY
  }

  override fun onDestroy() {
    stopped = true
    handler.removeCallbacksAndMessages(null)
    socket?.close(1000, null)
    socket = null
    super.onDestroy()
  }

  private fun connect() {
    val prefs = prefs(this)
    val base = prefs.getString(KEY_URL, null)
    val token = prefs.getString(KEY_TOKEN, null)
    if (base == null || token == null) {
      disable(this)
      return
    }
    val url = base.replaceFirst("http", "ws") + "/notifications/stream?token=" + token
    socket = client.newWebSocket(Request.Builder().url(url).build(), listener)
  }

  private fun scheduleReconnect() {
    if (stopped) return
    socket = null
    handler.postDelayed({ if (!stopped && socket == null) connect() }, retryMs)
    retryMs = minOf(retryMs * 2, MAX_RETRY_MS)
  }

  private val listener =
      object : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
          handler.post { retryMs = MIN_RETRY_MS }
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
          try {
            val event = JSONObject(text)
            if (event.optString("type") == "notify") {
              show(this@NotifyService, event.optString("title"), event.optString("body"))
            }
          } catch (_: Exception) {
            // ignore malformed frames
          }
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
          handler.post { if (socket === webSocket) scheduleReconnect() }
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
          handler.post {
            if (socket !== webSocket) return@post
            if (response?.code == 401) {
              // Signed out (or the session expired): stop for good until the
              // app turns this back on after the next sign-in.
              disable(this@NotifyService)
            } else {
              scheduleReconnect()
            }
          }
        }
      }

  companion object {
    const val CH_SERVICE = "superatlas-connection"
    const val CH_FRIENDS = "superatlas-friends"
    private const val NID_SERVICE = 0x5a01
    private const val MIN_RETRY_MS = 1000L
    private const val MAX_RETRY_MS = 60_000L
    private const val PREFS = "superatlas.notify"
    private const val KEY_ENABLED = "enabled"
    private const val KEY_URL = "url"
    private const val KEY_TOKEN = "token"

    private fun prefs(context: Context) = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun isEnabled(context: Context) = prefs(context).getBoolean(KEY_ENABLED, false)

    /** Turns background notifications on, connecting as this session. */
    fun enable(context: Context, apiUrl: String, token: String) {
      prefs(context).edit().putBoolean(KEY_ENABLED, true).putString(KEY_URL, apiUrl).putString(KEY_TOKEN, token).apply()
      start(context)
    }

    /** Turns them off and forgets the session. */
    fun disable(context: Context) {
      prefs(context).edit().clear().apply()
      context.stopService(Intent(context, NotifyService::class.java))
    }

    fun start(context: Context) {
      val intent = Intent(context, NotifyService::class.java)
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) context.startForegroundService(intent)
      else context.startService(intent)
    }

    fun ensureChannels(context: Context) {
      if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
      val nm = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
      nm.createNotificationChannel(
          NotificationChannel(CH_SERVICE, "Background connection", NotificationManager.IMPORTANCE_MIN).apply {
            description = "Keeps Super Atlas connected so friend requests reach you."
            setShowBadge(false)
          })
      nm.createNotificationChannel(
          NotificationChannel(CH_FRIENDS, "Friends", NotificationManager.IMPORTANCE_DEFAULT).apply {
            description = "Friend requests and new friends."
          })
    }

    private fun openApp(context: Context): PendingIntent =
        PendingIntent.getActivity(
            context,
            0,
            Intent(context, MainActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT)

    private fun show(context: Context, title: String, body: String) {
      val nm = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
      val n: Notification =
          NotificationCompat.Builder(context, CH_FRIENDS)
              .setSmallIcon(R.drawable.ic_notification)
              .setContentTitle(title)
              .setContentText(body)
              .setContentIntent(openApp(context))
              .setAutoCancel(true)
              .build()
      nm.notify((System.currentTimeMillis() % Int.MAX_VALUE).toInt(), n)
    }
  }
}
