package com.superatlasapp

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/** Restarts background notifications after a reboot or an app update, if
 *  they were on. */
class BootReceiver : BroadcastReceiver() {
  override fun onReceive(context: Context, intent: Intent) {
    if (intent.action != Intent.ACTION_BOOT_COMPLETED && intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) return
    if (NotifyService.isEnabled(context)) NotifyService.start(context)
  }
}
