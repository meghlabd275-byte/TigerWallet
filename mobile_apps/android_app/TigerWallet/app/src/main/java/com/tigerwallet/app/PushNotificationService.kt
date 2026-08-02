package com.tigerwallet.app

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * TigerWallet Push Notification Service
 * Production-ready push notifications for all wallet events
 */

class PushNotificationService : FirebaseMessagingService() {

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    companion object {
        const val CHANNEL_ID_TRANSACTIONS = "tigerwallet_transactions"
        const val CHANNEL_ID_SECURITY = "tigerwallet_security"
        const val CHANNEL_ID_PRICE_ALERTS = "tigerwallet_price_alerts"
        const val CHANNEL_ID_GENERAL = "tigerwallet_general"

        const val NOTIFICATION_TYPE_TRANSACTION = "transaction"
        const val NOTIFICATION_TYPE_SECURITY = "security"
        const val NOTIFICATION_TYPE_PRICE_ALERT = "price_alert"
        const val NOTIFICATION_TYPE_GENERAL = "general"
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannels()
    }

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // Send token to server for push notifications
        serviceScope.launch {
            sendTokenToServer(token)
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        super.onMessageReceived(message)

        val type = message.data["type"] ?: NOTIFICATION_TYPE_GENERAL
        val title = message.notification?.title ?: message.data["title"] ?: "TigerWallet"
        val body = message.notification?.body ?: message.data["body"] ?: ""

        when (type) {
            NOTIFICATION_TYPE_TRANSACTION -> showTransactionNotification(title, body, message.data)
            NOTIFICATION_TYPE_SECURITY -> showSecurityNotification(title, body, message.data)
            NOTIFICATION_TYPE_PRICE_ALERT -> showPriceAlertNotification(title, body, message.data)
            else -> showGeneralNotification(title, body, message.data)
        }
    }

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

            // Transaction notifications - high importance
            val transactionChannel = NotificationChannel(
                CHANNEL_ID_TRANSACTIONS,
                "Transactions",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Wallet transaction notifications"
                enableVibration(true)
                enableLights(true)
            }

            // Security notifications - critical importance
            val securityChannel = NotificationChannel(
                CHANNEL_ID_SECURITY,
                "Security Alerts",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Security and login alerts"
                enableVibration(true)
                enableLights(true)
            }

            // Price alerts
            val priceChannel = NotificationChannel(
                CHANNEL_ID_PRICE_ALERTS,
                "Price Alerts",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "Cryptocurrency price alerts"
            }

            // General notifications
            val generalChannel = NotificationChannel(
                CHANNEL_ID_GENERAL,
                "General",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "General notifications"
            }

            notificationManager.createNotificationChannels(
                listOf(transactionChannel, securityChannel, priceChannel, generalChannel)
            )
        }
    }

    private fun showTransactionNotification(title: String, body: String, data: Map<String, String>) {
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            putExtra("notification_type", NOTIFICATION_TYPE_TRANSACTION)
            putExtra("tx_hash", data["tx_hash"])
            putExtra("tx_status", data["tx_status"])
        }

        val pendingIntent = PendingIntent.getActivity(
            this,
            System.currentTimeMillis().toInt(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID_TRANSACTIONS)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setCategory(NotificationCompat.CATEGORY_TRANSACTION)
            .apply {
                if (data["tx_status"] == "confirmed") {
                    setColor(getColor(R.color.green_500))
                } else if (data["tx_status"] == "failed") {
                    setColor(getColor(R.color.red_500))
                }
            }
            .build()

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(System.currentTimeMillis().toInt(), notification)
    }

    private fun showSecurityNotification(title: String, body: String, data: Map<String, String>) {
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            putExtra("notification_type", NOTIFICATION_TYPE_SECURITY)
        }

        val pendingIntent = PendingIntent.getActivity(
            this,
            System.currentTimeMillis().toInt(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID_SECURITY)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setCategory(NotificationCompat.CATEGORY_ALARM)
            .setColor(getColor(R.color.orange_500))
            .build()

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(System.currentTimeMillis().toInt(), notification)
    }

    private fun showPriceAlertNotification(title: String, body: String, data: Map<String, String>) {
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            putExtra("notification_type", NOTIFICATION_TYPE_PRICE_ALERT)
            putExtra("token_symbol", data["token_symbol"])
        }

        val pendingIntent = PendingIntent.getActivity(
            this,
            System.currentTimeMillis().toInt(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID_PRICE_ALERTS)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(System.currentTimeMillis().toInt(), notification)
    }

    private fun showGeneralNotification(title: String, body: String, data: Map<String, String>) {
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        }

        val pendingIntent = PendingIntent.getActivity(
            this,
            System.currentTimeMillis().toInt(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID_GENERAL)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(System.currentTimeMillis().toInt(), notification)
    }

    private suspend fun sendTokenToServer(token: String) {
        try {
            // Send FCM token to backend
            val api = RetrofitClient.instance
            api.registerPushToken(PushTokenRequest(token))
        } catch (e: Exception) {
            // Handle error - store locally for retry
            LocalStorage.save("fcm_token", token)
        }
    }
}

// Data classes for API calls
data class PushTokenRequest(
    val token: String,
    val platform: String = "android",
    val app_version: String = BuildConfig.VERSION_NAME
)
