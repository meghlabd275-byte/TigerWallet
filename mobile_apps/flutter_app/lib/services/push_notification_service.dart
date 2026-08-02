import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

/// Push Notification Service for Flutter App
/// Production-ready push notifications
class PushNotificationService {
  static final PushNotificationService _instance = PushNotificationService._internal();
  factory PushNotificationService() => _instance;
  PushNotificationService._internal();

  final FlutterLocalNotificationsPlugin _notifications = FlutterLocalNotificationsPlugin();
  FirebaseMessaging? _firebaseMessaging;
  
  Function(String?)? onTokenRefresh;
  Function(Map<String, dynamic>)? onMessageReceived;
  
  /// Initialize the notification service
  Future<void> initialize() async {
    // Initialize local notifications
    const androidSettings = AndroidInitializationSettings('@mipmap/ic_launcher');
    const iosSettings = DarwinInitializationSettings(
      requestAlertPermission: true,
      requestBadgePermission: true,
      requestSoundPermission: true,
    );
    
    const initSettings = InitializationSettings(
      android: androidSettings,
      iOS: iosSettings,
    );
    
    await _notifications.initialize(
      initSettings,
      onDidReceiveNotificationResponse: _onNotificationTapped,
    );
    
    // Initialize Firebase (if available)
    try {
      _firebaseMessaging = FirebaseMessaging.instance;
      await _firebaseMessaging?.requestPermission(
        alert: true,
        announcement: false,
        badge: true,
        carPlay: false,
        criticalAlert: false,
        provisional: false,
        sound: true,
      );
      
      // Get FCM token
      final token = await _firebaseMessaging?.getToken();
      debugPrint('FCM Token: $token');
      
      // Handle token refresh
      _firebaseMessaging?.onTokenRefresh.listen((token) {
        debugPrint('FCM Token refreshed: $token');
        onTokenRefresh?.call(token);
      });
      
      // Handle foreground messages
      _firebaseMessaging?.onMessage.listen((message) {
        debugPrint('Received foreground message: ${message.notification?.title}');
        onMessageReceived?.call(message.data);
      });
    } catch (e) {
      debugPrint('Firebase not available: $e');
    }
  }
  
  /// Request notification permissions
  Future<bool> requestPermission() async {
    if (Platform.isIOS) {
      final result = await _notifications
          .resolvePlatformSpecificImplementation<IOSFlutterLocalNotificationsPlugin>()
          ?.requestPermissions(
            alert: true,
            badge: true,
            sound: true,
          );
      return result ?? false;
    } else if (Platform.isAndroid) {
      final android = _notifications.resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin>();
      final result = await android?.requestNotificationsPermission();
      return result ?? false;
    }
    return false;
  }
  
  /// Get FCM token
  Future<String?> getToken() async {
    try {
      return await _firebaseMessaging?.getToken();
    } catch (e) {
      debugPrint('Error getting FCM token: $e');
      return null;
    }
  }
  
  /// Subscribe to topic
  Future<void> subscribeToTopic(String topic) async {
    try {
      await _firebaseMessaging?.subscribeToTopic(topic);
      debugPrint('Subscribed to topic: $topic');
    } catch (e) {
      debugPrint('Error subscribing to topic: $e');
    }
  }
  
  /// Unsubscribe from topic
  Future<void> unsubscribeFromTopic(String topic) async {
    try {
      await _firebaseMessaging?.unsubscribeFromTopic(topic);
      debugPrint('Unsubscribed from topic: $topic');
    } catch (e) {
      debugPrint('Error unsubscribing from topic: $e');
    }
  }
  
  /// Show local notification
  Future<void> showNotification({
    required int id,
    required String title,
    required String body,
    String? payload,
  }) async {
    const androidDetails = AndroidNotificationDetails(
      'tigerwallet_channel',
      'TigerWallet Notifications',
      channelDescription: 'Notifications from TigerWallet',
      importance: Importance.high,
      priority: Priority.high,
      icon: '@mipmap/ic_launcher',
    );
    
    const iosDetails = DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: true,
      presentSound: true,
    );
    
    const details = NotificationDetails(
      android: androidDetails,
      iOS: iosDetails,
    );
    
    await _notifications.show(id, title, body, details, payload: payload);
  }
  
  /// Show transaction notification
  Future<void> showTransactionNotification({
    required String txHash,
    required String status,
    required String amount,
    required String symbol,
  }) async {
    final title = status == 'confirmed' ? 'Transaction Confirmed' : 'Transaction Sent';
    final body = status == 'confirmed' 
        ? '$amount $symbol has been confirmed!'
        : 'Your transaction of $amount $symbol is being processed.';
    
    await showNotification(
      id: txHash.hashCode,
      title: title,
      body: body,
      payload: txHash,
    );
  }
  
  /// Show price alert notification
  Future<void> showPriceAlertNotification({
    required String token,
    required double price,
    required bool isAbove,
  }) async {
    final direction = isAbove ? 'above' : 'below';
    final title = 'Price Alert: $token';
    final body = '$token is now $direction \$${price.toStringAsFixed(2)}';
    
    await showNotification(
      id: '${token}_price_$price'.hashCode,
      title: title,
      body: body,
    );
  }
  
  /// Show security alert
  Future<void> showSecurityAlert({
    required String title,
    required String message,
  }) async {
    await showNotification(
      id: DateTime.now().millisecondsSinceEpoch,
      title: '⚠️ Security Alert',
      body: message,
    );
  }
  
  /// Show NFT activity notification
  Future<void> showNFTNotification({
    required String action,
    required String nftName,
    required String from,
  }) async {
    final title = 'NFT $action';
    final body = '$nftName $action from $from';
    
    await showNotification(
      id: DateTime.now().millisecondsSinceEpoch,
      title: title,
      body: body,
    );
  }
  
  /// Cancel notification
  Future<void> cancelNotification(int id) async {
    await _notifications.cancel(id);
  }
  
  /// Cancel all notifications
  Future<void> cancelAllNotifications() async {
    await _notifications.cancelAll();
  }
  
  /// Handle notification tap
  void _onNotificationTapped(NotificationResponse response) {
    debugPrint('Notification tapped: ${response.payload}');
    // Handle navigation based on payload
    if (response.payload != null) {
      // Navigate to transaction details or other screens
    }
  }
  
  /// Get initial message (when app launched from notification)
  Future<RemoteMessage?> getInitialMessage() async {
    try {
      return await _firebaseMessaging?.getInitialMessage();
    } catch (e) {
      debugPrint('Error getting initial message: $e');
      return null;
    }
  }
}

// Notification Types
enum NotificationType {
  transaction,
  priceAlert,
  security,
  nft,
  staking,
  general,
}

// Extension for notification type
extension NotificationTypeExtension on NotificationType {
  String get channelId {
    switch (this) {
      case NotificationType.transaction:
        return 'tigerwallet_transactions';
      case NotificationType.priceAlert:
        return 'tigerwallet_price_alerts';
      case NotificationType.security:
        return 'tigerwallet_security';
      case NotificationType.nft:
        return 'tigerwallet_nft';
      case NotificationType.staking:
        return 'tigerwallet_staking';
      case NotificationType.general:
        return 'tigerwallet_general';
    }
  }
  
  String get channelName {
    switch (this) {
      case NotificationType.transaction:
        return 'Transactions';
      case NotificationType.priceAlert:
        return 'Price Alerts';
      case NotificationType.security:
        return 'Security Alerts';
      case NotificationType.nft:
        return 'NFT Activity';
      case NotificationType.staking:
        return 'Staking';
      case NotificationType.general:
        return 'General';
    }
  }
}
