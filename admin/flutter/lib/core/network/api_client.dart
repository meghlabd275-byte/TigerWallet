/**
 * TigerWallet Admin - API Client
 * Complete HTTP Client with all endpoints
 */

import 'dart:convert';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../constants/app_constants.dart';
import 'logger.dart';

abstract class ApiClient {
  // Auth
  Future<Map<String, dynamic>> login(String email, String password);
  Future<void> logout();
  Future<Map<String, dynamic>> refreshToken(String refreshToken);
  Future<void> setup2FA();
  Future<bool> verify2FA(String code);
  Future<void> changePassword(String oldPassword, String newPassword);
  
  // Admins
  Future<List<Map<String, dynamic>>> getAdmins();
  Future<Map<String, dynamic>> getAdmin(String id);
  Future<Map<String, dynamic>> createAdmin(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateAdmin(String id, Map<String, dynamic> data);
  Future<void> deleteAdmin(String id);
  Future<void> suspendAdmin(String id);
  Future<void> activateAdmin(String id);
  
  // Users
  Future<Map<String, dynamic>> getUsers({int page = 1, int pageSize = 20, String? search, String? status});
  Future<Map<String, dynamic>> getUser(String id);
  Future<Map<String, dynamic>> updateUser(String id, Map<String, dynamic> data);
  Future<void> banUser(String id);
  Future<void> unbanUser(String id);
  Future<void> suspendUser(String id);
  
  // KYC
  Future<Map<String, dynamic>> getKycRequests({int page = 1, int pageSize = 20, String? status});
  Future<Map<String, dynamic>> getKyc(String id);
  Future<void> approveKyc(String id);
  Future<void> rejectKyc(String id, String reason);
  
  // Transactions
  Future<Map<String, dynamic>> getTransactions({int page = 1, int pageSize = 20, String? status, String? userId});
  Future<Map<String, dynamic>> getTransaction(String id);
  Future<void> flagTransaction(String id, String reason);
  Future<void> unflagTransaction(String id);
  
  // Withdrawals
  Future<Map<String, dynamic>> getWithdrawals({int page = 1, int pageSize = 20, String? status});
  Future<Map<String, dynamic>> getWithdrawal(String id);
  Future<void> approveWithdrawal(String id);
  Future<void> rejectWithdrawal(String id, String reason);
  Future<void> processWithdrawal(String id);
  
  // Tokens
  Future<Map<String, dynamic>> getTokens({int page = 1, int pageSize = 20, String? search, String? status});
  Future<Map<String, dynamic>> getToken(String id);
  Future<Map<String, dynamic>> createToken(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateToken(String id, Map<String, dynamic> data);
  Future<void> deleteToken(String id);
  Future<void> verifyToken(String id);
  
  // Pairs
  Future<Map<String, dynamic>> getPairs({int page = 1, int pageSize = 20, String? status});
  Future<Map<String, dynamic>> getPair(String id);
  Future<Map<String, dynamic>> createPair(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updatePair(String id, Map<String, dynamic> data);
  Future<void> haltPair(String id);
  Future<void> activatePair(String id);
  
  // Blockchains
  Future<List<Map<String, dynamic>>> getBlockchains();
  Future<Map<String, dynamic>> getBlockchain(String id);
  Future<Map<String, dynamic>> createBlockchain(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateBlockchain(String id, Map<String, dynamic> data);
  
  // Fees
  Future<List<Map<String, dynamic>>> getFees({int? chainId});
  Future<Map<String, dynamic>> createFee(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateFee(String id, Map<String, dynamic> data);
  
  // White Labels
  Future<Map<String, dynamic>> getWhiteLabels({int page = 1, int pageSize = 20, String? status});
  Future<Map<String, dynamic>> getWhiteLabel(String id);
  Future<Map<String, dynamic>> createWhiteLabel(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateWhiteLabel(String id, Map<String, dynamic> data);
  Future<void> activateWhiteLabel(String id);
  Future<void> suspendWhiteLabel(String id);
  
  // Tickets
  Future<Map<String, dynamic>> getTickets({int page = 1, int pageSize = 20, String? status, String? priority});
  Future<Map<String, dynamic>> getTicket(String id);
  Future<Map<String, dynamic>> createTicket(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateTicket(String id, Map<String, dynamic> data);
  Future<void> assignTicket(String id, String adminId);
  Future<void> addTicketMessage(String id, String message);
  
  // Analytics
  Future<Map<String, dynamic>> getDashboardStats();
  Future<Map<String, dynamic>> getUserAnalytics({String? startDate, String? endDate});
  Future<Map<String, dynamic>> getTransactionAnalytics({String? startDate, String? endDate});
  Future<Map<String, dynamic>> getRevenueAnalytics({String? startDate, String? endDate});
  Future<List<Map<String, dynamic>>> getVolumeChart({String? startDate, String? endDate, String? interval});
  
  // Audit
  Future<Map<String, dynamic>> getAuditLogs({int page = 1, int pageSize = 50, String? adminId, String? action});
  Future<String> exportAuditLogs(Map<String, dynamic> filters);
  
  // Feature Flags
  Future<List<Map<String, dynamic>>> getFeatureFlags();
  Future<Map<String, dynamic>> createFeatureFlag(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateFeatureFlag(String id, Map<String, dynamic> data);
  Future<void> deleteFeatureFlag(String id);
  
  // Notifications
  Future<List<Map<String, dynamic>>> getNotifications();
  Future<void> markNotificationRead(String id);
  Future<void> broadcastNotification(String title, String message, String type);
  
  // Backups
  Future<List<Map<String, dynamic>>> getBackups();
  Future<Map<String, dynamic>> createBackup(String type);
  Future<void> restoreBackup(String id);
  Future<void> deleteBackup(String id);
  
  // Webhooks
  Future<List<Map<String, dynamic>>> getWebhooks();
  Future<Map<String, dynamic>> createWebhook(Map<String, dynamic> data);
  Future<Map<String, dynamic>> updateWebhook(String id, Map<String, dynamic> data);
  Future<void> testWebhook(String id);
  Future<void> deleteWebhook(String id);

  // Admin platform endpoints (wallet_api :8443 /api/v1/admin/*)
  Future<List<Map<String, dynamic>>> getAdminWallets();
  Future<Map<String, dynamic>> getAdminStats();
  Future<List<Map<String, dynamic>>> getAdminCryptoCards({String? status});
  Future<void> blockCryptoCard(String id);
  Future<void> activateCryptoCard(String id);
  Future<List<Map<String, dynamic>>> getAdminFeatureFlags();
  Future<Map<String, dynamic>> createFeatureFlag2(Map<String, dynamic> data);
  Future<void> toggleFeatureFlag(String id);
  Future<List<Map<String, dynamic>>> getAdminLiquidityPools();
  Future<Map<String, dynamic>> getAdminLiquidityStats();
  Future<void> addLiquidity(String poolId, Map<String, dynamic> data);
  Future<List<Map<String, dynamic>>> getAdminP2PMerchants({String? status});
  Future<void> approveP2PMerchant(String id);
  Future<void> rejectP2PMerchant(String id, String reason);
  Future<List<Map<String, dynamic>>> getAdminMarginPositions();
  Future<Map<String, dynamic>> getAdminMarginLiquidationStats();
  Future<void> liquidateMarginPosition(String id);
  Future<void> transferMasterWallet(String walletId, String toAddress, double amount);
}
