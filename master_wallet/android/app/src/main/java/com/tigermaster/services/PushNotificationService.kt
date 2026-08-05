/**
 * PushNotificationService - Android Implementation
 * Firebase Cloud Messaging for Master Wallet
 */

package com.tigermaster.services

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import org.json.JSONObject

class PushNotificationService : FirebaseMessagingService() {
    
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    
    companion object {
        private const val TAG = "PushNotificationService"
        
        // Notification channels
        const val CHANNEL_TRANSACTIONS = "transactions"
        const val CHANNEL_BALANCE = "balance"
        const val CHANNEL_SECURITY = "security"
        const val CHANNEL_ALERTS = "alerts"
        
        // Notification IDs
        const val NOTIFICATION_ID_TRANSACTION = 1001
        const val NOTIFICATION_ID_BALANCE = 1002
        const val NOTIFICATION_ID_SECURITY = 1003
        const val NOTIFICATION_ID_ALERT = 1004
        
        private var fcmToken: String? = null
        
        fun getFcmToken(): String? = fcmToken
    }
    
    override fun onCreate() {
        super.onCreate()
        createNotificationChannels()
    }
    
    override fun onNewToken(token: String) {
        super.onNewToken(token)
        Log.d(TAG, "New FCM token: $token")
        fcmToken = token
        
        // Send token to server
        scope.launch {
            sendTokenToServer(token)
        }
    }
    
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)
        Log.d(TAG, "Message received from: ${remoteMessage.from}")
        
        // Check notification payload
        remoteMessage.notification?.let { notification ->
            Log.d(TAG, "Notification title: ${notification.title}")
            Log.d(TAG, "Notification body: ${notification.body}")
        }
        
        // Check data payload
        if (remoteMessage.data.isNotEmpty()) {
            Log.d(TAG, "Data payload: ${remoteMessage.data}")
            handleDataMessage(remoteMessage.data)
        }
    }
    
    /**
     * Create notification channels for Android O+
     */
    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = getSystemService(NotificationManager::class.java)
            
            // Transaction channel
            val transactionChannel = NotificationChannel(
                CHANNEL_TRANSACTIONS,
                "Transactions",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Notifications for incoming and outgoing transactions"
                enableVibration(true)
            }
            
            // Balance channel
            val balanceChannel = NotificationChannel(
                CHANNEL_BALANCE,
                "Balance Updates",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "Notifications for balance changes"
            }
            
            // Security channel
            val securityChannel = NotificationChannel(
                CHANNEL_SECURITY,
                "Security Alerts",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Important security alerts"
                enableVibration(true)
            }
            
            // Alerts channel
            val alertsChannel = NotificationChannel(
                CHANNEL_ALERTS,
                "Price Alerts",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "Price alerts and market notifications"
            }
            
            notificationManager.createNotificationChannels(
                listOf(
                    transactionChannel,
                    balanceChannel,
                    securityChannel,
                    alertsChannel
                )
            )
        }
    }
    
    /**
     * Handle data message from server
     */
    private fun handleDataMessage(data: Map<String, String>) {
        val type = data["type"] ?: return
        
        when (type) {
            "transaction" -> handleTransactionNotification(data)
            "balance_update" -> handleBalanceNotification(data)
            "security_alert" -> handleSecurityNotification(data)
            "price_alert" -> handlePriceAlert(data)
            "approval_request" -> handleApprovalRequest(data)
            "kyc_update" -> handleKycUpdate(data)
        }
    }
    
    /**
     * Handle transaction notification
     */
    private fun handleTransactionNotification(data: Map<String, String>) {
        val title = data["title"] ?: "Transaction Update"
        val body = data["body"] ?: ""
        val txHash = data["txHash"] ?: ""
        val amount = data["amount"] ?: ""
        val token = data["token"] ?: "ETH"
        val status = data["status"] ?: "pending"
        
        showNotification(
            channelId = CHANNEL_TRANSACTIONS,
            notificationId = NOTIFICATION_ID_TRANSACTION,
            title = title,
            body = body,
            data = mapOf("txHash" to txHash, "type" to "transaction")
        )
    }
    
    /**
     * Handle balance update notification
     */
    private fun handleBalanceNotification(data: Map<String, String>) {
        val title = data["title"] ?: "Balance Update"
        val body = data["body"] ?: ""
        val chainId = data["chainId"] ?: "1"
        
        showNotification(
            channelId = CHANNEL_BALANCE,
            notificationId = NOTIFICATION_ID_BALANCE,
            title = title,
            body = body,
            data = mapOf("chainId" to chainId, "type" to "balance")
        )
    }
    
    /**
     * Handle security alert
     */
    private fun handleSecurityNotification(data: Map<String, String>) {
        val title = data["title"] ?: "Security Alert"
        val body = data["body"] ?: ""
        
        showNotification(
            channelId = CHANNEL_SECURITY,
            notificationId = NOTIFICATION_ID_SECURITY,
            title = title,
            body = body,
            data = mapOf("type" to "security")
        )
    }
    
    /**
     * Handle price alert
     */
    private fun handlePriceAlert(data: Map<String, String>) {
        val title = data["title"] ?: "Price Alert"
        val body = data["body"] ?: ""
        val pair = data["pair"] ?: ""
        val price = data["price"] ?: ""
        
        showNotification(
            channelId = CHANNEL_ALERTS,
            notificationId = NOTIFICATION_ID_ALERT,
            title = title,
            body = body,
            data = mapOf("pair" to pair, "price" to price, "type" to "price_alert")
        )
    }
    
    /**
     * Handle approval request (for multi-sig)
     */
    private fun handleApprovalRequest(data: Map<String, String>) {
        val title = data["title"] ?: "Approval Required"
        val body = data["body"] ?: ""
        val txId = data["txId"] ?: ""
        
        showNotification(
            channelId = CHANNEL_SECURITY,
            notificationId = NOTIFICATION_ID_SECURITY,
            title = title,
            body = body,
            data = mapOf("txId" to txId, "type" to "approval")
        )
    }
    
    /**
     * Handle KYC update
     */
    private fun handleKycUpdate(data: Map<String, String>) {
        val title = data["title"] ?: "KYC Update"
        val body = data["body"] ?: ""
        
        showNotification(
            channelId = CHANNEL_ALERTS,
            notificationId = NOTIFICATION_ID_ALERT,
            title = title,
            body = body,
            data = mapOf("type" to "kyc")
        )
    }
    
    /**
     * Show notification
     */
    private fun showNotification(
        channelId: String,
        notificationId: Int,
        title: String,
        body: String,
        data: Map<String, String>
    ) {
        val notificationManager = getSystemService(NotificationManager::class.java)
        
        // Create intent for notification tap
        val intent = Intent(this, NotificationReceiver::class.java).apply {
            putExtra("data", JSONObject(data).toString())
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        }
        
        val pendingIntent = PendingIntent.getBroadcast(
            this,
            notificationId,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        
        val builder = android.app.Notification.Builder(this, channelId)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setPriority(Notification.PRIORITY_HIGH)
        
        notificationManager.notify(notificationId, builder.build())
    }
    
    /**
     * Send FCM token to server
     */
    private suspend fun sendTokenToServer(token: String) {
        try {
            // In production, call your server API
            Log.d(TAG, "Token sent to server: $token")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send token to server: ${e.message}")
        }
    }
    
    /**
     * Get FCM token (call this from activity)
     */
    fun getToken() {
        FirebaseMessaging.getInstance().token.addOnCompleteListener { task ->
            if (!task.isSuccessful) {
                Log.w(TAG, "Fetching FCM token failed", task.exception)
                return@addOnCompleteListener
            }
            
            fcmToken = task.result
            Log.d(TAG, "FCM Token: $fcmToken")
            
            // Send to server
            scope.launch {
                sendTokenToServer(fcmToken!!)
            }
        }
    }
    
    /**
     * Subscribe to topic
     */
    fun subscribeToTopic(topic: String) {
        FirebaseMessaging.getInstance().subscribeToTopic(topic)
            .addOnCompleteListener { task ->
                if (task.isSuccessful) {
                    Log.d(TAG, "Subscribed to topic: $topic")
                } else {
                    Log.e(TAG, "Failed to subscribe to topic: $topic")
                }
            }
    }
    
    /**
     * Unsubscribe from topic
     */
    fun unsubscribeFromTopic(topic: String) {
        FirebaseMessaging.getInstance().unsubscribeFromTopic(topic)
            .addOnCompleteListener { task ->
                if (task.isSuccessful) {
                    Log.d(TAG, "Unsubscribed from topic: $topic")
                } else {
                    Log.e(TAG, "Failed to unsubscribe from topic: $topic")
                }
            }
    }
}

/**
 * Notification receiver for handling notification taps
 */
class NotificationReceiver : android.content.BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val dataStr = intent.getStringExtra("data")
        if (dataStr != null) {
            try {
                val data = JSONObject(dataStr)
                val type = data.optString("type")
                
                // Handle notification tap based on type
                when (type) {
                    "transaction" -> {
                        val txHash = data.optString("txHash")
                        // Navigate to transaction details
                        Log.d("NotificationReceiver", "Transaction: $txHash")
                    }
                    "balance" -> {
                        // Navigate to balance
                        Log.d("NotificationReceiver", "Balance update")
                    }
                    "security" -> {
                        // Navigate to security center
                        Log.d("NotificationReceiver", "Security alert")
                    }
                    "approval" -> {
                        // Navigate to approval
                        Log.d("NotificationReceiver", "Approval request")
                    }
                }
            } catch (e: Exception) {
                Log.e("NotificationReceiver", "Error parsing data: ${e.message}")
            }
        }
    }
}
