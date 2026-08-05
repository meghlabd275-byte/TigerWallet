/**
 * TigerWallet Admin Platform - Internationalization (i18n)
 * Multi-language support for global users
 * Supports: English, Spanish, Chinese, Japanese, Korean, French, German, Portuguese, Russian, Arabic, Hindi
 */

import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

// ============================================================================
// Types & Interfaces
// ============================================================================

export type Language = 'en' | 'es' | 'zh' | 'ja' | 'ko' | 'fr' | 'de' | 'pt' | 'ru' | 'ar' | 'hi';

export interface LanguageOption {
  code: Language;
  name: string;
  nativeName: string;
  flag: string;
  rtl?: boolean;
}

export interface TranslationContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: (key: string, params?: Record<string, string | number>) => string;
  languages: LanguageOption[];
  isRTL: boolean;
}

// ============================================================================
// Language Options
// ============================================================================

export const LANGUAGES: LanguageOption[] = [
  { code: 'en', name: 'English', nativeName: 'English', flag: '🇺🇸' },
  { code: 'es', name: 'Spanish', nativeName: 'Español', flag: '🇪🇸' },
  { code: 'zh', name: 'Chinese', nativeName: '中文', flag: '🇨🇳' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語', flag: '🇯🇵' },
  { code: 'ko', name: 'Korean', nativeName: '한국어', flag: '🇰🇷' },
  { code: 'fr', name: 'French', nativeName: 'Français', flag: '🇫🇷' },
  { code: 'de', name: 'German', nativeName: 'Deutsch', flag: '🇩🇪' },
  { code: 'pt', name: 'Portuguese', nativeName: 'Português', flag: '🇧🇷' },
  { code: 'ru', name: 'Russian', nativeName: 'Русский', flag: '🇷🇺' },
  { code: 'ar', name: 'Arabic', nativeName: 'العربية', flag: '🇸🇦', rtl: true },
  { code: 'hi', name: 'Hindi', nativeName: 'हिन्दी', flag: '🇮🇳' },
];

// ============================================================================
// Translation Dictionaries
// ============================================================================

const translations: Record<Language, Record<string, string>> = {
  en: {
    // Navigation
    'nav.dashboard': 'Dashboard',
    'nav.users': 'Users',
    'nav.wallets': 'Wallets',
    'nav.transactions': 'Transactions',
    'nav.blockchains': 'Blockchains',
    'nav.tokens': 'Tokens',
    'nav.pairs': 'Trading Pairs',
    'nav.fees': 'Fees',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'White Label',
    'nav.settings': 'Settings',
    'nav.logout': 'Logout',
    'nav.auditLogs': 'Audit Logs',
    'nav.reports': 'Reports',
    'nav.batchOperations': 'Batch Operations',
    
    // Dashboard
    'dashboard.title': 'Admin Dashboard',
    'dashboard.totalUsers': 'Total Users',
    'dashboard.activeUsers': 'Active Users',
    'dashboard.pendingKYC': 'Pending KYC',
    'dashboard.totalTransactions': 'Total Transactions',
    'dashboard.volume24h': '24h Volume',
    'dashboard.revenue24h': '24h Revenue',
    
    // Users
    'users.title': 'User Management',
    'users.search': 'Search users...',
    'users.email': 'Email',
    'users.wallet': 'Wallet Address',
    'users.status': 'Status',
    'users.kyc': 'KYC Status',
    'users.actions': 'Actions',
    'users.suspend': 'Suspend',
    'users.ban': 'Ban',
    'users.view': 'View Details',
    
    // Transactions
    'transactions.title': 'Transaction Management',
    'transactions.hash': 'Transaction Hash',
    'transactions.type': 'Type',
    'transactions.amount': 'Amount',
    'transactions.status': 'Status',
    'transactions.timestamp': 'Timestamp',
    
    // Common
    'common.save': 'Save',
    'common.cancel': 'Cancel',
    'common.delete': 'Delete',
    'common.edit': 'Edit',
    'common.create': 'Create',
    'common.search': 'Search',
    'common.filter': 'Filter',
    'common.export': 'Export',
    'common.import': 'Import',
    'common.refresh': 'Refresh',
    'common.loading': 'Loading...',
    'common.noData': 'No data available',
    'common.success': 'Success',
    'common.error': 'Error',
    'common.confirm': 'Confirm',
    'common.back': 'Back',
    'common.next': 'Next',
    'common.previous': 'Previous',
    'common.submit': 'Submit',
    'common.close': 'Close',
    'common.download': 'Download',
    'common.upload': 'Upload',
    
    // Status
    'status.active': 'Active',
    'status.inactive': 'Inactive',
    'status.pending': 'Pending',
    'status.suspended': 'Suspended',
    'status.banned': 'Banned',
    'status.completed': 'Completed',
    'status.failed': 'Failed',
    'status.processing': 'Processing',
    
    // Auth
    'auth.login': 'Login',
    'auth.logout': 'Logout',
    'auth.email': 'Email',
    'auth.password': 'Password',
    'auth.2fa': 'Two-Factor Code',
    'auth.rememberMe': 'Remember me',
    'auth.forgotPassword': 'Forgot password?',
    
    // Theme
    'theme.title': 'Theme',
    'theme.light': 'Light',
    'theme.dark': 'Dark',
    'theme.system': 'System',
    
    // Language
    'language.title': 'Language',
    'language.select': 'Select Language',
  },
  
  es: {
    'nav.dashboard': 'Panel',
    'nav.users': 'Usuarios',
    'nav.wallets': 'Billeteras',
    'nav.transactions': 'Transacciones',
    'nav.blockchains': 'Blockchains',
    'nav.tokens': 'Tokens',
    'nav.pairs': 'Pares de Trading',
    'nav.fees': 'Tarifas',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'White Label',
    'nav.settings': 'Configuración',
    'nav.logout': 'Cerrar Sesión',
    'nav.auditLogs': 'Registros de Auditoría',
    'nav.reports': 'Informes',
    'nav.batchOperations': 'Operaciones por Lotes',
    
    'dashboard.title': 'Panel de Administración',
    'dashboard.totalUsers': 'Total de Usuarios',
    'dashboard.activeUsers': 'Usuarios Activos',
    'dashboard.pendingKYC': 'KYC Pendiente',
    'dashboard.totalTransactions': 'Total de Transacciones',
    'dashboard.volume24h': 'Volumen 24h',
    'dashboard.revenue24h': 'Ingresos 24h',
    
    'users.title': 'Gestión de Usuarios',
    'users.search': 'Buscar usuarios...',
    'users.email': 'Correo',
    'users.wallet': 'Dirección de Billetera',
    'users.status': 'Estado',
    'users.kyc': 'Estado KYC',
    'users.actions': 'Acciones',
    'users.suspend': 'Suspender',
    'users.ban': 'Bloquear',
    'users.view': 'Ver Detalles',
    
    'transactions.title': 'Gestión de Transacciones',
    'transactions.hash': 'Hash de Transacción',
    'transactions.type': 'Tipo',
    'transactions.amount': 'Monto',
    'transactions.status': 'Estado',
    'transactions.timestamp': 'Marca de Tiempo',
    
    'common.save': 'Guardar',
    'common.cancel': 'Cancelar',
    'common.delete': 'Eliminar',
    'common.edit': 'Editar',
    'common.create': 'Crear',
    'common.search': 'Buscar',
    'common.filter': 'Filtrar',
    'common.export': 'Exportar',
    'common.import': 'Importar',
    'common.refresh': 'Actualizar',
    'common.loading': 'Cargando...',
    'common.noData': 'Sin datos disponibles',
    'common.success': 'Éxito',
    'common.error': 'Error',
    'common.confirm': 'Confirmar',
    'common.back': 'Atrás',
    'common.next': 'Siguiente',
    'common.previous': 'Anterior',
    'common.submit': 'Enviar',
    'common.close': 'Cerrar',
    'common.download': 'Descargar',
    'common.upload': 'Subir',
    
    'status.active': 'Activo',
    'status.inactive': 'Inactivo',
    'status.pending': 'Pendiente',
    'status.suspended': 'Suspendido',
    'status.banned': 'Bloqueado',
    'status.completed': 'Completado',
    'status.failed': 'Fallido',
    'status.processing': 'Procesando',
    
    'auth.login': 'Iniciar Sesión',
    'auth.logout': 'Cerrar Sesión',
    'auth.email': 'Correo',
    'auth.password': 'Contraseña',
    'auth.2fa': 'Código 2FA',
    'auth.rememberMe': 'Recordarme',
    'auth.forgotPassword': '¿Olvidaste tu contraseña?',
    
    'theme.title': 'Tema',
    'theme.light': 'Claro',
    'theme.dark': 'Oscuro',
    'theme.system': 'Sistema',
    
    'language.title': 'Idioma',
    'language.select': 'Seleccionar Idioma',
  },
  
  zh: {
    'nav.dashboard': '仪表板',
    'nav.users': '用户',
    'nav.wallets': '钱包',
    'nav.transactions': '交易',
    'nav.blockchains': '区块链',
    'nav.tokens': '代币',
    'nav.pairs': '交易对',
    'nav.fees': '费用',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': '白标',
    'nav.settings': '设置',
    'nav.logout': '退出',
    'nav.auditLogs': '审计日志',
    'nav.reports': '报告',
    'nav.batchOperations': '批量操作',
    
    'dashboard.title': '管理仪表板',
    'dashboard.totalUsers': '总用户数',
    'dashboard.activeUsers': '活跃用户',
    'dashboard.pendingKYC': '待处理KYC',
    'dashboard.totalTransactions': '总交易数',
    'dashboard.volume24h': '24小时交易量',
    'dashboard.revenue24h': '24小时收入',
    
    'users.title': '用户管理',
    'users.search': '搜索用户...',
    'users.email': '邮箱',
    'users.wallet': '钱包地址',
    'users.status': '状态',
    'users.kyc': 'KYC状态',
    'users.actions': '操作',
    'users.suspend': '暂停',
    'users.ban': '封禁',
    'users.view': '查看详情',
    
    'transactions.title': '交易管理',
    'transactions.hash': '交易哈希',
    'transactions.type': '类型',
    'transactions.amount': '金额',
    'transactions.status': '状态',
    'transactions.timestamp': '时间戳',
    
    'common.save': '保存',
    'common.cancel': '取消',
    'common.delete': '删除',
    'common.edit': '编辑',
    'common.create': '创建',
    'common.search': '搜索',
    'common.filter': '筛选',
    'common.export': '导出',
    'common.import': '导入',
    'common.refresh': '刷新',
    'common.loading': '加载中...',
    'common.noData': '暂无数据',
    'common.success': '成功',
    'common.error': '错误',
    'common.confirm': '确认',
    'common.back': '返回',
    'common.next': '下一步',
    'common.previous': '上一步',
    'common.submit': '提交',
    'common.close': '关闭',
    'common.download': '下载',
    'common.upload': '上传',
    
    'status.active': '活跃',
    'status.inactive': '未激活',
    'status.pending': '待处理',
    'status.suspended': '已暂停',
    'status.banned': '已封禁',
    'status.completed': '已完成',
    'status.failed': '失败',
    'status.processing': '处理中',
    
    'auth.login': '登录',
    'auth.logout': '退出',
    'auth.email': '邮箱',
    'auth.password': '密码',
    'auth.2fa': '两步验证码',
    'auth.rememberMe': '记住我',
    'auth.forgotPassword': '忘记密码？',
    
    'theme.title': '主题',
    'theme.light': '浅色',
    'theme.dark': '深色',
    'theme.system': '跟随系统',
    
    'language.title': '语言',
    'language.select': '选择语言',
  },
  
  ja: {
    'nav.dashboard': 'ダッシュボード',
    'nav.users': 'ユーザー',
    'nav.wallets': 'ウォレット',
    'nav.transactions': '取引',
    'nav.blockchains': 'ブロックチェーン',
    'nav.tokens': 'トークン',
    'nav.pairs': '取引ペア',
    'nav.fees': '手数料',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'ホワイトラベル',
    'nav.settings': '設定',
    'nav.logout': 'ログアウト',
    'nav.auditLogs': '監査ログ',
    'nav.reports': 'レポート',
    'nav.batchOperations': 'バッチ操作',
    
    'dashboard.title': '管理ダッシュボード',
    'dashboard.totalUsers': '総ユーザー数',
    'dashboard.activeUsers': 'アクティブユーザー',
    'dashboard.pendingKYC': '保留中のKYC',
    'dashboard.totalTransactions': '総取引数',
    'dashboard.volume24h': '24時間取引量',
    'dashboard.revenue24h': '24時間収益',
    
    'users.title': 'ユーザー管理',
    'users.search': 'ユーザーを検索...',
    'users.email': 'メールアドレス',
    'users.wallet': 'ウォレット住所',
    'users.status': 'ステータス',
    'users.kyc': 'KYCステータス',
    'users.actions': 'アクション',
    'users.suspend': '一時停止',
    'users.ban': 'BAN',
    'users.view': '詳細を見る',
    
    'transactions.title': '取引管理',
    'transactions.hash': 'トランザクションハッシュ',
    'transactions.type': 'タイプ',
    'transactions.amount': '金額',
    'transactions.status': 'ステータス',
    'transactions.timestamp': 'タイムスタンプ',
    
    'common.save': '保存',
    'common.cancel': 'キャンセル',
    'common.delete': '削除',
    'common.edit': '編集',
    'common.create': '作成',
    'common.search': '検索',
    'common.filter': 'フィルター',
    'common.export': 'エクスポート',
    'common.import': 'インポート',
    'common.refresh': '更新',
    'common.loading': '読み込み中...',
    'common.noData': 'データがありません',
    'common.success': '成功',
    'common.error': 'エラー',
    'common.confirm': '確認',
    'common.back': '戻る',
    'common.next': '次へ',
    'common.previous': '前へ',
    'common.submit': '送信',
    'common.close': '閉じる',
    'common.download': 'ダウンロード',
    'common.upload': 'アップロード',
    
    'status.active': 'アクティブ',
    'status.inactive': '非アクティブ',
    'status.pending': '保留中',
    'status.suspended': '一時停止',
    'status.banned': 'BAN',
    'status.completed': '完了',
    'status.failed': '失敗',
    'status.processing': '処理中',
    
    'auth.login': 'ログイン',
    'auth.logout': 'ログアウト',
    'auth.email': 'メール',
    'auth.password': 'パスワード',
    'auth.2fa': '2FAコード',
    'auth.rememberMe': 'ログイン状態を保持',
    'auth.forgotPassword': 'パスワードを忘れた？',
    
    'theme.title': 'テーマ',
    'theme.light': 'ライト',
    'theme.dark': 'ダーク',
    'theme.system': 'システム',
    
    'language.title': '言語',
    'language.select': '言語を選択',
  },
  
  ko: {
    'nav.dashboard': '대시보드',
    'nav.users': '사용자',
    'nav.wallets': '지갑',
    'nav.transactions': '거래',
    'nav.blockchains': '블록체인',
    'nav.tokens': '토큰',
    'nav.pairs': '거래쌍',
    'nav.fees': '수수료',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': '화이트 라벨',
    'nav.settings': '설정',
    'nav.logout': '로그아웃',
    'nav.auditLogs': '감사 로그',
    'nav.reports': '보고서',
    'nav.batchOperations': '일괄 작업',
    
    'dashboard.title': '관리 대시보드',
    'dashboard.totalUsers': '총 사용자',
    'dashboard.activeUsers': '활성 사용자',
    'dashboard.pendingKYC': '대기중 KYC',
    'dashboard.totalTransactions': '총 거래',
    'dashboard.volume24h': '24시간 거래량',
    'dashboard.revenue24h': '24시간 수익',
    
    'users.title': '사용자 관리',
    'users.search': '사용자 검색...',
    'users.email': '이메일',
    'users.wallet': '지갑 주소',
    'users.status': '상태',
    'users.kyc': 'KYC 상태',
    'users.actions': '작업',
    'users.suspend': '정지',
    'users.ban': '차단',
    'users.view': '상세 보기',
    
    'transactions.title': '거래 관리',
    'transactions.hash': '트랜잭션 해시',
    'transactions.type': '유형',
    'transactions.amount': '금액',
    'transactions.status': '상태',
    'transactions.timestamp': '타임스탬프',
    
    'common.save': '저장',
    'common.cancel': '취소',
    'common.delete': '삭제',
    'common.edit': '편집',
    'common.create': '생성',
    'common.search': '검색',
    'common.filter': '필터',
    'common.export': '내보내기',
    'common.import': '가져오기',
    'common.refresh': '새로고침',
    'common.loading': '로딩중...',
    'common.noData': '데이터 없음',
    'common.success': '성공',
    'common.error': '오류',
    'common.confirm': '확인',
    'common.back': '뒤로',
    'common.next': '다음',
    'common.previous': '이전',
    'common.submit': '제출',
    'common.close': '닫기',
    'common.download': '다운로드',
    'common.upload': '업로드',
    
    'status.active': '활성',
    'status.inactive': '비활성',
    'status.pending': '대기중',
    'status.suspended': '정지됨',
    'status.banned': '차단됨',
    'status.completed': '완료',
    'status.failed': '실패',
    'status.processing': '처리중',
    
    'auth.login': '로그인',
    'auth.logout': '로그아웃',
    'auth.email': '이메일',
    'auth.password': '비밀번호',
    'auth.2fa': '2FA 코드',
    'auth.rememberMe': '로그인 유지',
    'auth.forgotPassword': '비밀번호 찾기',
    
    'theme.title': '테마',
    'theme.light': '라이트',
    'theme.dark': '다크',
    'theme.system': '시스템',
    
    'language.title': '언어',
    'language.select': '언어 선택',
  },
  
  fr: {
    'nav.dashboard': 'Tableau de bord',
    'nav.users': 'Utilisateurs',
    'nav.wallets': 'Portefeuilles',
    'nav.transactions': 'Transactions',
    'nav.blockchains': 'Blockchains',
    'nav.tokens': 'Jetons',
    'nav.pairs': 'Paires de trading',
    'nav.fees': 'Frais',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'White Label',
    'nav.settings': 'Paramètres',
    'nav.logout': 'Déconnexion',
    'nav.auditLogs': 'Journaux d\'audit',
    'nav.reports': 'Rapports',
    'nav.batchOperations': 'Opérations par lots',
    
    'dashboard.title': 'Tableau de bord admin',
    'dashboard.totalUsers': 'Total utilisateurs',
    'dashboard.activeUsers': 'Utilisateurs actifs',
    'dashboard.pendingKYC': 'KYC en attente',
    'dashboard.totalTransactions': 'Total transactions',
    'dashboard.volume24h': 'Volume 24h',
    'dashboard.revenue24h': 'Revenus 24h',
    
    'users.title': 'Gestion des utilisateurs',
    'users.search': 'Rechercher...',
    'users.email': 'Email',
    'users.wallet': 'Adresse portefeuille',
    'users.status': 'Statut',
    'users.kyc': 'Statut KYC',
    'users.actions': 'Actions',
    'users.suspend': 'Suspendre',
    'users.ban': 'Bannir',
    'users.view': 'Voir détails',
    
    'transactions.title': 'Gestion des transactions',
    'transactions.hash': 'Hash de transaction',
    'transactions.type': 'Type',
    'transactions.amount': 'Montant',
    'transactions.status': 'Statut',
    'transactions.timestamp': 'Horodatage',
    
    'common.save': 'Enregistrer',
    'common.cancel': 'Annuler',
    'common.delete': 'Supprimer',
    'common.edit': 'Modifier',
    'common.create': 'Créer',
    'common.search': 'Rechercher',
    'common.filter': 'Filtrer',
    'common.export': 'Exporter',
    'common.import': 'Importer',
    'common.refresh': 'Actualiser',
    'common.loading': 'Chargement...',
    'common.noData': 'Aucune donnée',
    'common.success': 'Succès',
    'common.error': 'Erreur',
    'common.confirm': 'Confirmer',
    'common.back': 'Retour',
    'common.next': 'Suivant',
    'common.previous': 'Précédent',
    'common.submit': 'Soumettre',
    'common.close': 'Fermer',
    'common.download': 'Télécharger',
    'common.upload': 'Téléverser',
    
    'status.active': 'Actif',
    'status.inactive': 'Inactif',
    'status.pending': 'En attente',
    'status.suspended': 'Suspendu',
    'status.banned': 'Banni',
    'status.completed': 'Terminé',
    'status.failed': 'Échoué',
    'status.processing': 'En cours',
    
    'auth.login': 'Connexion',
    'auth.logout': 'Déconnexion',
    'auth.email': 'Email',
    'auth.password': 'Mot de passe',
    'auth.2fa': 'Code 2FA',
    'auth.rememberMe': 'Se souvenir',
    'auth.forgotPassword': 'Mot de passe oublié?',
    
    'theme.title': 'Thème',
    'theme.light': 'Clair',
    'theme.dark': 'Sombre',
    'theme.system': 'Système',
    
    'language.title': 'Langue',
    'language.select': 'Sélectionner la langue',
  },
  
  de: {
    'nav.dashboard': 'Dashboard',
    'nav.users': 'Benutzer',
    'nav.wallets': 'Wallets',
    'nav.transactions': 'Transaktionen',
    'nav.blockchains': 'Blockchains',
    'nav.tokens': 'Tokens',
    'nav.pairs': 'Handelspaare',
    'nav.fees': 'Gebühren',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'White Label',
    'nav.settings': 'Einstellungen',
    'nav.logout': 'Abmelden',
    'nav.auditLogs': 'Audit-Protokolle',
    'nav.reports': 'Berichte',
    'nav.batchOperations': 'Stapelverarbeitung',
    
    'dashboard.title': 'Admin-Dashboard',
    'dashboard.totalUsers': 'Benutzer gesamt',
    'dashboard.activeUsers': 'Aktive Benutzer',
    'dashboard.pendingKYC': 'Ausstehende KYC',
    'dashboard.totalTransactions': 'Transaktionen gesamt',
    'dashboard.volume24h': '24h Volumen',
    'dashboard.revenue24h': '24h Einnahmen',
    
    'users.title': 'Benutzerverwaltung',
    'users.search': 'Benutzer suchen...',
    'users.email': 'E-Mail',
    'users.wallet': 'Wallet-Adresse',
    'users.status': 'Status',
    'users.kyc': 'KYC-Status',
    'users.actions': 'Aktionen',
    'users.suspend': 'Suspendieren',
    'users.ban': 'Sperren',
    'users.view': 'Details anzeigen',
    
    'transactions.title': 'Transaktionsverwaltung',
    'transactions.hash': 'Transaktions-Hash',
    'transactions.type': 'Typ',
    'transactions.amount': 'Betrag',
    'transactions.status': 'Status',
    'transactions.timestamp': 'Zeitstempel',
    
    'common.save': 'Speichern',
    'common.cancel': 'Abbrechen',
    'common.delete': 'Löschen',
    'common.edit': 'Bearbeiten',
    'common.create': 'Erstellen',
    'common.search': 'Suchen',
    'common.filter': 'Filtern',
    'common.export': 'Exportieren',
    'common.import': 'Importieren',
    'common.refresh': 'Aktualisieren',
    'common.loading': 'Laden...',
    'common.noData': 'Keine Daten',
    'common.success': 'Erfolg',
    'common.error': 'Fehler',
    'common.confirm': 'Bestätigen',
    'common.back': 'Zurück',
    'common.next': 'Weiter',
    'common.previous': 'Vorherige',
    'common.submit': 'Absenden',
    'common.close': 'Schließen',
    'common.download': 'Herunterladen',
    'common.upload': 'Hochladen',
    
    'status.active': 'Aktiv',
    'status.inactive': 'Inaktiv',
    'status.pending': 'Ausstehend',
    'status.suspended': 'Suspendiert',
    'status.banned': 'Gesperrt',
    'status.completed': 'Abgeschlossen',
    'status.failed': 'Fehlgeschlagen',
    'status.processing': 'Verarbeitung',
    
    'auth.login': 'Anmelden',
    'auth.logout': 'Abmelden',
    'auth.email': 'E-Mail',
    'auth.password': 'Passwort',
    'auth.2fa': '2FA-Code',
    'auth.rememberMe': 'Angemeldet bleiben',
    'auth.forgotPassword': 'Passwort vergessen?',
    
    'theme.title': 'Thema',
    'theme.light': 'Hell',
    'theme.dark': 'Dunkel',
    'theme.system': 'System',
    
    'language.title': 'Sprache',
    'language.select': 'Sprache wählen',
  },
  
  pt: {
    'nav.dashboard': 'Painel',
    'nav.users': 'Usuários',
    'nav.wallets': 'Carteiras',
    'nav.transactions': 'Transações',
    'nav.blockchains': 'Blockchains',
    'nav.tokens': 'Tokens',
    'nav.pairs': 'Pares de trading',
    'nav.fees': 'Taxas',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'White Label',
    'nav.settings': 'Configurações',
    'nav.logout': 'Sair',
    'nav.auditLogs': 'Logs de auditoria',
    'nav.reports': 'Relatórios',
    'nav.batchOperations': 'Operações em lote',
    
    'dashboard.title': 'Painel de admin',
    'dashboard.totalUsers': 'Total de usuários',
    'dashboard.activeUsers': 'Usuários ativos',
    'dashboard.pendingKYC': 'KYC pendente',
    'dashboard.totalTransactions': 'Total de transações',
    'dashboard.volume24h': 'Volume 24h',
    'dashboard.revenue24h': 'Receita 24h',
    
    'users.title': 'Gerenciamento de usuários',
    'users.search': 'Buscar usuários...',
    'users.email': 'Email',
    'users.wallet': 'Endereço da carteira',
    'users.status': 'Status',
    'users.kyc': 'Status KYC',
    'users.actions': 'Ações',
    'users.suspend': 'Suspender',
    'users.ban': 'Banir',
    'users.view': 'Ver detalhes',
    
    'transactions.title': 'Gerenciamento de transações',
    'transactions.hash': 'Hash da transação',
    'transactions.type': 'Tipo',
    'transactions.amount': 'Quantia',
    'transactions.status': 'Status',
    'transactions.timestamp': 'Timestamp',
    
    'common.save': 'Salvar',
    'common.cancel': 'Cancelar',
    'common.delete': 'Excluir',
    'common.edit': 'Editar',
    'common.create': 'Criar',
    'common.search': 'Buscar',
    'common.filter': 'Filtrar',
    'common.export': 'Exportar',
    'common.import': 'Importar',
    'common.refresh': 'Atualizar',
    'common.loading': 'Carregando...',
    'common.noData': 'Sem dados',
    'common.success': 'Sucesso',
    'common.error': 'Erro',
    'common.confirm': 'Confirmar',
    'common.back': 'Voltar',
    'common.next': 'Próximo',
    'common.previous': 'Anterior',
    'common.submit': 'Enviar',
    'common.close': 'Fechar',
    'common.download': 'Baixar',
    'common.upload': 'Enviar',
    
    'status.active': 'Ativo',
    'status.inactive': 'Inativo',
    'status.pending': 'Pendente',
    'status.suspended': 'Suspenso',
    'status.banned': 'Banido',
    'status.completed': 'Concluído',
    'status.failed': 'Falhou',
    'status.processing': 'Processando',
    
    'auth.login': 'Entrar',
    'auth.logout': 'Sair',
    'auth.email': 'Email',
    'auth.password': 'Senha',
    'auth.2fa': 'Código 2FA',
    'auth.rememberMe': 'Lembrar',
    'auth.forgotPassword': 'Esqueceu a senha?',
    
    'theme.title': 'Tema',
    'theme.light': 'Claro',
    'theme.dark': 'Escuro',
    'theme.system': 'Sistema',
    
    'language.title': 'Idioma',
    'language.select': 'Selecionar idioma',
  },
  
  ru: {
    'nav.dashboard': 'Панель',
    'nav.users': 'Пользователи',
    'nav.wallets': 'Кошельки',
    'nav.transactions': 'Транзакции',
    'nav.blockchains': 'Блокчейны',
    'nav.tokens': 'Токены',
    'nav.pairs': 'Торговые пары',
    'nav.fees': 'Комиссии',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'White Label',
    'nav.settings': 'Настройки',
    'nav.logout': 'Выйти',
    'nav.auditLogs': 'Журналы аудита',
    'nav.reports': 'Отчеты',
    'nav.batchOperations': 'Пакетные операции',
    
    'dashboard.title': 'Админ панель',
    'dashboard.totalUsers': 'Всего пользователей',
    'dashboard.activeUsers': 'Активные пользователи',
    'dashboard.pendingKYC': 'KYC на рассмотрении',
    'dashboard.totalTransactions': 'Всего транзакций',
    'dashboard.volume24h': 'Объём за 24ч',
    'dashboard.revenue24h': 'Доход за 24ч',
    
    'users.title': 'Управление пользователями',
    'users.search': 'Поиск пользователей...',
    'users.email': 'Email',
    'users.wallet': 'Адрес кошелька',
    'users.status': 'Статус',
    'users.kyc': 'Статус KYC',
    'users.actions': 'Действия',
    'users.suspend': 'Приостановить',
    'users.ban': 'Заблокировать',
    'users.view': 'Подробнее',
    
    'transactions.title': 'Управление транзакциями',
    'transactions.hash': 'Хэш транзакции',
    'transactions.type': 'Тип',
    'transactions.amount': 'Сумма',
    'transactions.status': 'Статус',
    'transactions.timestamp': 'Время',
    
    'common.save': 'Сохранить',
    'common.cancel': 'Отмена',
    'common.delete': 'Удалить',
    'common.edit': 'Редактировать',
    'common.create': 'Создать',
    'common.search': 'Поиск',
    'common.filter': 'Фильтр',
    'common.export': 'Экспорт',
    'common.import': 'Импорт',
    'common.refresh': 'Обновить',
    'common.loading': 'Загрузка...',
    'common.noData': 'Нет данных',
    'common.success': 'Успех',
    'common.error': 'Ошибка',
    'common.confirm': 'Подтвердить',
    'common.back': 'Назад',
    'common.next': 'Далее',
    'common.previous': 'Назад',
    'common.submit': 'Отправить',
    'common.close': 'Закрыть',
    'common.download': 'Скачать',
    'common.upload': 'Загрузить',
    
    'status.active': 'Активен',
    'status.inactive': 'Неактивен',
    'status.pending': 'Ожидание',
    'status.suspended': 'Приостановлен',
    'status.banned': 'Заблокирован',
    'status.completed': 'Завершено',
    'status.failed': 'Ошибка',
    'status.processing': 'Обработка',
    
    'auth.login': 'Войти',
    'auth.logout': 'Выйти',
    'auth.email': 'Email',
    'auth.password': 'Пароль',
    'auth.2fa': 'Код 2FA',
    'auth.rememberMe': 'Запомнить',
    'auth.forgotPassword': 'Забыли пароль?',
    
    'theme.title': 'Тема',
    'theme.light': 'Светлая',
    'theme.dark': 'Тёмная',
    'theme.system': 'Системная',
    
    'language.title': 'Язык',
    'language.select': 'Выбрать язык',
  },
  
  ar: {
    'nav.dashboard': 'لوحة التحكم',
    'nav.users': 'المستخدمون',
    'nav.wallets': 'المحافظ',
    'nav.transactions': 'المعاملات',
    'nav.blockchains': 'البلوكتشين',
    'nav.tokens': 'الرموز',
    'nav.pairs': 'أزواج التداول',
    'nav.fees': 'الرسوم',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'العلامة البيضاء',
    'nav.settings': 'الإعدادات',
    'nav.logout': 'تسجيل الخروج',
    'nav.auditLogs': 'سجلات التدقيق',
    'nav.reports': 'التقارير',
    'nav.batchOperations': 'العمليات الجماعية',
    
    'dashboard.title': 'لوحة تحكم المسؤول',
    'dashboard.totalUsers': 'إجمالي المستخدمين',
    'dashboard.activeUsers': 'المستخدمون النشطون',
    'dashboard.pendingKYC': 'KYC المعلق',
    'dashboard.totalTransactions': 'إجمالي المعاملات',
    'dashboard.volume24h': 'حجم 24 ساعة',
    'dashboard.revenue24h': 'الإيرادات 24 ساعة',
    
    'users.title': 'إدارة المستخدمين',
    'users.search': 'البحث عن المستخدمين...',
    'users.email': 'البريد الإلكتروني',
    'users.wallet': 'عنوان المحفظة',
    'users.status': 'الحالة',
    'users.kyc': 'حالة KYC',
    'users.actions': 'الإجراءات',
    'users.suspend': 'تعليق',
    'users.ban': 'حظر',
    'users.view': 'عرض التفاصيل',
    
    'transactions.title': 'إدارة المعاملات',
    'transactions.hash': 'تجزئة المعاملة',
    'transactions.type': 'النوع',
    'transactions.amount': 'المبلغ',
    'transactions.status': 'الحالة',
    'transactions.timestamp': 'الطابع الزمني',
    
    'common.save': 'حفظ',
    'common.cancel': 'إلغاء',
    'common.delete': 'حذف',
    'common.edit': 'تعديل',
    'common.create': 'إنشاء',
    'common.search': 'بحث',
    'common.filter': 'تصفية',
    'common.export': 'تصدير',
    'common.import': 'استيراد',
    'common.refresh': 'تحديث',
    'common.loading': 'جاري التحميل...',
    'common.noData': 'لا توجد بيانات',
    'common.success': 'نجاح',
    'common.error': 'خطأ',
    'common.confirm': 'تأكيد',
    'common.back': 'رجوع',
    'common.next': 'التالي',
    'common.previous': 'السابق',
    'common.submit': 'إرسال',
    'common.close': 'إغلاق',
    'common.download': 'تحميل',
    'common.upload': 'رفع',
    
    'status.active': 'نشط',
    'status.inactive': 'غير نشط',
    'status.pending': 'معلق',
    'status.suspended': 'موقوف',
    'status.banned': 'محظور',
    'status.completed': 'مكتمل',
    'status.failed': 'فشل',
    'status.processing': 'قيد المعالجة',
    
    'auth.login': 'تسجيل الدخول',
    'auth.logout': 'تسجيل الخروج',
    'auth.email': 'البريد الإلكتروني',
    'auth.password': 'كلمة المرور',
    'auth.2fa': 'رمز 2FA',
    'auth.rememberMe': 'تذكرني',
    'auth.forgotPassword': 'نسيت كلمة المرور؟',
    
    'theme.title': 'السمة',
    'theme.light': 'فاتح',
    'theme.dark': 'داكن',
    'theme.system': 'النظام',
    
    'language.title': 'اللغة',
    'language.select': 'اختر اللغة',
  },
  
  hi: {
    'nav.dashboard': 'डैशबोर्ड',
    'nav.users': 'उपयोगकर्ता',
    'nav.wallets': 'वॉलेट',
    'nav.transactions': 'लेन-देन',
    'nav.blockchains': 'ब्लॉकचेन',
    'nav.tokens': 'टोकन',
    'nav.pairs': 'ट्रेडिंग पेयर',
    'nav.fees': 'शुल्क',
    'nav.kyc': 'KYC',
    'nav.whiteLabel': 'व्हाइट लेबल',
    'nav.settings': 'सेटिंग्स',
    'nav.logout': 'लॉग आउट',
    'nav.auditLogs': 'ऑडिट लॉग',
    'nav.reports': 'रिपोर्ट',
    'nav.batchOperations': 'बैच ऑपरेशन',
    
    'dashboard.title': 'एडमिन डैशबोर्ड',
    'dashboard.totalUsers': 'कुल उपयोगकर्ता',
    'dashboard.activeUsers': 'सक्रिय उपयोगकर्ता',
    'dashboard.pendingKYC': 'लंबित KYC',
    'dashboard.totalTransactions': 'कुल लेन-देन',
    'dashboard.volume24h': '24 घंटे वॉल्यूम',
    'dashboard.revenue24h': '24 घंटे राजस्व',
    
    'users.title': 'उपयोगकर्ता प्रबंधन',
    'users.search': 'उपयोगकर्ता खोजें...',
    'users.email': 'ईमेल',
    'users.wallet': 'वॉलेट पता',
    'users.status': 'स्थिति',
    'users.kyc': 'KYC स्थिति',
    'users.actions': 'कार्रवाई',
    'users.suspend': 'निलंबित',
    'users.ban': 'प्रतिबंध',
    'users.view': 'विवरण देखें',
    
    'transactions.title': 'लेन-देन प्रबंधन',
    'transactions.hash': 'लेन-देन हैश',
    'transactions.type': 'प्रकार',
    'transactions.amount': 'राशि',
    'transactions.status': 'स्थिति',
    'transactions.timestamp': 'समय',
    
    'common.save': 'सहेजें',
    'common.cancel': 'रद्द करें',
    'common.delete': 'हटाएं',
    'common.edit': 'संपादित करें',
    'common.create': 'बनाएं',
    'common.search': 'खोजें',
    'common.filter': 'फ़िल्टर',
    'common.export': 'निर्यात',
    'common.import': 'आयात',
    'common.refresh': 'रीफ्रेश',
    'common.loading': 'लोड हो रहा है...',
    'common.noData': 'कोई डेटा नहीं',
    'common.success': 'सफल',
    'common.error': 'त्रुटि',
    'common.confirm': 'पुष्टि करें',
    'common.back': 'वापस',
    'common.next': 'आगे',
    'common.previous': 'पिछला',
    'common.submit': 'सबमिट',
    'common.close': 'बंद करें',
    'common.download': 'डाउनलोड',
    'common.upload': 'अपलोड',
    
    'status.active': 'सक्रिय',
    'status.inactive': 'निष्क्रिय',
    'status.pending': 'लंबित',
    'status.suspended': 'निलंबित',
    'status.banned': 'प्रतिबंधित',
    'status.completed': 'पूर्ण',
    'status.failed': 'विफल',
    'status.processing': 'प्रसंस्करण',
    
    'auth.login': 'लॉग इन',
    'auth.logout': 'लॉग आउट',
    'auth.email': 'ईमेल',
    'auth.password': 'पासवर्ड',
    'auth.2fa': '2FA कोड',
    'auth.rememberMe': 'याद रखें',
    'auth.forgotPassword': 'पासवर्ड भूल गए?',
    
    'theme.title': 'थीम',
    'theme.light': 'लाइट',
    'theme.dark': 'डार्क',
    'theme.system': 'सिस्टम',
    
    'language.title': 'भाषा',
    'language.select': 'भाषा चुनें',
  },
};

// ============================================================================
// Context
// ============================================================================

const TranslationContext = createContext<TranslationContextType | undefined>(undefined);

// ============================================================================
// Provider
// ============================================================================

interface TranslationProviderProps {
  children: ReactNode;
}

export const TranslationProvider: React.FC<TranslationProviderProps> = ({ children }) => {
  const [language, setLanguageState] = useState<Language>(() => {
    // Check localStorage first
    const savedLang = localStorage.getItem('admin_language') as Language;
    if (savedLang && LANGUAGES.some(l => l.code === savedLang)) {
      return savedLang;
    }
    // Check browser language
    const browserLang = navigator.language.split('-')[0];
    if (LANGUAGES.some(l => l.code === browserLang)) {
      return browserLang as Language;
    }
    return 'en';
  });

  const setLanguage = useCallback((lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem('admin_language', lang);
    
    // Set RTL for Arabic
    if (lang === 'ar') {
      document.documentElement.dir = 'rtl';
    } else {
      document.documentElement.dir = 'ltr';
    }
  }, []);

  const t = useCallback((key: string, params?: Record<string, string | number>): string => {
    const langTranslations = translations[language] || translations.en;
    let text = langTranslations[key] || translations.en[key] || key;
    
    if (params) {
      Object.entries(params).forEach(([paramKey, paramValue]) => {
        text = text.replace(new RegExp(`{{${paramKey}}}`, 'g'), String(paramValue));
      });
    }
    
    return text;
  }, [language]);

  const isRTL = language === 'ar';

  const value: TranslationContextType = {
    language,
    setLanguage,
    t,
    languages: LANGUAGES,
    isRTL,
  };

  return (
    <TranslationContext.Provider value={value}>
      {children}
    </TranslationContext.Provider>
  );
};

// ============================================================================
// Hook
// ============================================================================

export const useTranslation = (): TranslationContextType => {
  const context = useContext(TranslationContext);
  if (!context) {
    throw new Error('useTranslation must be used within a TranslationProvider');
  }
  return context;
};

export default TranslationProvider;
