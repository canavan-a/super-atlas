package com.superatlasapp

import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

/**
 * Shows the server's FCM notifications while the app is open (Android only
 * shows "notification" messages itself while it's in the background). New
 * tokens are picked up by JS on its next launch (src/notifications.ts).
 */
class PlayMessagingService : FirebaseMessagingService() {
  override fun onMessageReceived(message: RemoteMessage) {
    val n = message.notification ?: return
    PushModule.ensureChannel(this)
    val open =
        PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT)
    val nm = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
    nm.notify(
        (System.currentTimeMillis() % Int.MAX_VALUE).toInt(),
        NotificationCompat.Builder(this, PushModule.CHANNEL)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(n.title)
            .setContentText(n.body)
            .setContentIntent(open)
            .setAutoCancel(true)
            .build())
  }

  override fun onNewToken(token: String) {}
}
