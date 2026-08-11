/**
 * TigerWallet Admin - App Constants
 */

class AppConstants {
  // App Info
  static const String appName = 'TigerWallet Admin';
  static const String appVersion = '1.0.0';
  
  // API
  static const String baseUrl = 'http://localhost:8443/api/v1';
  static const String wsUrl = '';
  static const int apiTimeout = 30000;
  static const int wsTimeout = 5000;
  
  // Storage Keys
  static const String themeKey = 'app_theme';
  static const String tokenKey = 'auth_token';
  static const String refreshTokenKey = 'refresh_token';
  static const String userKey = 'user_data';
  static const String settingsKey = 'app_settings';
  
  // Routes
  static const String splashRoute = '/splash';
  static const String loginRoute = '/login';
  static const String dashboardRoute = '/dashboard';
  static const String usersRoute = '/users';
  static const String kycRoute = '/kyc';
  static const String transactionsRoute = '/transactions';
  static const String tokensRoute = '/tokens';
  static const String pairsRoute = '/pairs';
  static const String blockchainsRoute = '/blockchains';
  static const String whitelabelsRoute = '/whitelabels';
  static const String ticketsRoute = '/tickets';
  static const String analyticsRoute = '/analytics';
  static const String settingsRoute = '/settings';
  static const String moreRoute = '/more';
  
  // Pagination
  static const int defaultPageSize = 20;
  static const int maxPageSize = 100;
  
  // Animation Durations
  static const Duration shortAnimation = Duration(milliseconds: 200);
  static const Duration mediumAnimation = Duration(milliseconds: 300);
  static const Duration longAnimation = Duration(milliseconds: 500);
  
  // Cache
  static const Duration cacheExpiration = Duration(minutes: 5);
  static const int maxCacheSize = 100;
  
  // Validation
  static const int minPasswordLength = 8;
  static const int maxPasswordLength = 128;
  static const int minUsernameLength = 3;
  static const int maxUsernameLength = 50;
  
  // Regex Patterns
  static final RegExp emailRegex = RegExp(
    r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$',
  );
  static final RegExp phoneRegex = RegExp(r'^\+?[1-9]\d{1,14}$');
  static final RegExp walletAddressRegex = RegExp(r'^0x[a-fA-F0-9]{40}$');
}

// API Endpoints
class ApiEndpoints {
  // Auth
  static const String login = '/auth/login';
  static const String logout = '/auth/logout';
  static const String refresh = '/auth/refresh';
  static const String register = '/auth/register';
  static const String forgotPassword = '/auth/forgot-password';
  static const String resetPassword = '/auth/reset-password';
  static const String verifyEmail = '/auth/verify-email';
  static const String setup2FA = '/auth/2fa/setup';
  static const String verify2FA = '/auth/2fa/verify';
  
  // Admins
  static const String admins = '/admins';
  static const String adminProfile = '/admins/profile';
  
  // Users
  static const String users = '/users';
  static const String userDetail = '/users/{id}';
  static const String userUpdate = '/users/{id}';
  static const String userBan = '/users/{id}/ban';
  static const String userUnban = '/users/{id}/unban';
  static const String userSuspend = '/users/{id}/suspend';
  
  // KYC
  static const String kycRequests = '/kyc';
  static const String kycApprove = '/kyc/{id}/approve';
  static const String kycReject = '/kyc/{id}/reject';
  
  // Transactions
  static const String transactions = '/transactions';
  static const String transactionDetail = '/transactions/{id}';
  static const String transactionFlag = '/transactions/{id}/flag';
  static const String transactionUnflag = '/transactions/{id}/unflag';
  
  // Withdrawals
  static const String withdrawals = '/withdrawals';
  static const String withdrawalApprove = '/withdrawals/{id}/approve';
  static const String withdrawalReject = '/withdrawals/{id}/reject';
  static const String withdrawalProcess = '/withdrawals/{id}/process';
  
  // Tokens
  static const String tokens = '/tokens';
  static const String tokenDetail = '/tokens/{id}';
  static const String tokenCreate = '/tokens';
  static const String tokenUpdate = '/tokens/{id}';
  static const String tokenDelete = '/tokens/{id}';
  static const String tokenVerify = '/tokens/{id}/verify';
  
  // Pairs
  static const String pairs = '/pairs';
  static const String pairDetail = '/pairs/{id}';
  static const String pairCreate = '/pairs';
  static const String pairUpdate = '/pairs/{id}';
  static const String pairHalt = '/pairs/{id}/halt';
  static const String pairActivate = '/pairs/{id}/activate';
  
  // Blockchains
  static const String blockchains = '/blockchains';
  static const String blockchainDetail = '/blockchains/{id}';
  static const String blockchainCreate = '/blockchains';
  static const String blockchainUpdate = '/blockchains/{id}';
  
  // Fees
  static const String fees = '/fees';
  static const String feeCreate = '/fees';
  static const String feeUpdate = '/fees/{id}';
  
  // White Labels
  static const String whiteLabels = '/whitelabels';
  static const String whiteLabelDetail = '/whitelabels/{id}';
  static const String whiteLabelCreate = '/whitelabels';
  static const String whiteLabelUpdate = '/whitelabels/{id}';
  static const String whiteLabelActivate = '/whitelabels/{id}/activate';
  static const String whiteLabelSuspend = '/whitelabels/{id}/suspend';
  
  // Tickets
  static const String tickets = '/tickets';
  static const String ticketDetail = '/tickets/{id}';
  static const String ticketCreate = '/tickets';
  static const String ticketUpdate = '/tickets/{id}';
  static const String ticketAssign = '/tickets/{id}/assign';
  static const String ticketMessage = '/tickets/{id}/messages';
  
  // Analytics
  static const String analyticsDashboard = '/analytics/dashboard';
  static const String analyticsUsers = '/analytics/users';
  static const String analyticsTransactions = '/analytics/transactions';
  static const String analyticsRevenue = '/analytics/revenue';
  
  // Audit
  static const String auditLogs = '/audit-logs';
  static const String auditExport = '/audit-logs/export';
  
  // Feature Flags
  static const String featureFlags = '/feature-flags';
  static const String featureFlagCreate = '/feature-flags';
  static const String featureFlagUpdate = '/feature-flags/{id}';
  
  // Notifications
  static const String notifications = '/notifications';
  static const String notificationRead = '/notifications/{id}/read';
  static const String notificationBroadcast = '/notifications/broadcast';
  
  // Backups
  static const String backups = '/backups';
  static const String backupCreate = '/backups';
  static const String backupRestore = '/backups/{id}/restore';
  static const String backupDelete = '/backups/{id}';
  
  // Webhooks
  static const String webhooks = '/webhooks';
  static const String webhookCreate = '/webhooks';
  static const String webhookUpdate = '/webhooks/{id}';
  static const String webhookTest = '/webhooks/{id}/test';
  static const String webhookDelete = '/webhooks/{id}';
  
  // Dashboard
  static const String dashboard = '/dashboard';
  static const String stats = '/stats';
}
