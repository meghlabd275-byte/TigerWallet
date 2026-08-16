// TigerWallet Flutter — White-label branding config
//
// Loads WL branding for a rebranded Flutter build:
//   1. `--dart-define=WL_BRANDING_SLUG=xxx` (set per WL-client build).
//      Absent/empty => stock TigerWallet build; no remote fetch happens.
//   2. If a slug is set, fetch `GET {WL_CONTROL_PLANE_URL}/api/v1/branding/{slug}`
//      on app startup (via the `http` package, already a dep). The endpoint is
//      PUBLIC (no auth) so a WL-branded app needs no secrets.
//   3. Cache the fetched JSON in SharedPreferences so a transient network
//      failure / cold start still shows the WL brand instead of TigerWallet.
//   4. Fall back to TigerWallet defaults if there is no slug, the fetch fails,
//      or the endpoint returns 404 (no WL client matches the slug).
//
// BrandingConfig is a [ChangeNotifier]; ThemeProvider listens to it and
// rebuilds the Material theme with the WL primary/secondary colors whenever
// the branding refreshes.

import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

/// TigerWallet stock branding — the backward-compatible default.
class Branding {
  final String slug;
  final String appName;
  final String logoUrl;
  final String primaryColor;
  final String secondaryColor;
  final String domain;
  final String supportEmail;
  final String termsUrl;
  final String privacyUrl;

  const Branding({
    this.slug = '',
    this.appName = 'TigerWallet',
    this.logoUrl = '',
    this.primaryColor = '#FF6B35',
    this.secondaryColor = '#F7C94B',
    this.domain = 'tigerwallet.io',
    this.supportEmail = 'support@tigerwallet.io',
    this.termsUrl = 'https://tigerwallet.io/terms',
    this.privacyUrl = 'https://tigerwallet.io/privacy',
  });

  /// TigerWallet stock branding.
  static const Branding defaults = Branding();

  /// Merge a JSON map from the control plane over the TigerWallet defaults.
  /// Missing/empty fields fall back to defaults (never null/empty).
  factory Branding.fromJson(Map<String, dynamic> json, {String slug = ''}) {
    String pick(String key, String dflt) {
      final v = json[key];
      if (v is String && v.trim().isNotEmpty) return v.trim();
      return dflt;
    }

    return Branding(
      slug: slug,
      appName: pick('app_name', defaults.appName),
      logoUrl: pick('logo_url', defaults.logoUrl),
      primaryColor: pick('primary_color', defaults.primaryColor),
      secondaryColor: pick('secondary_color', defaults.secondaryColor),
      domain: pick('domain', defaults.domain),
      supportEmail: pick('support_email', defaults.supportEmail),
      termsUrl: pick('terms_url', defaults.termsUrl),
      privacyUrl: pick('privacy_url', defaults.privacyUrl),
    );
  }

  Map<String, dynamic> toJson() => {
        'slug': slug,
        'app_name': appName,
        'logo_url': logoUrl,
        'primary_color': primaryColor,
        'secondary_color': secondaryColor,
        'domain': domain,
        'support_email': supportEmail,
        'terms_url': termsUrl,
        'privacy_url': privacyUrl,
      };

  Branding copyWith({String? slug}) =>
      Branding(slug: slug ?? this.slug);

  @override
  String toString() => 'Branding(slug=$slug, appName=$appName, domain=$domain)';
}

/// Singleton white-label branding config. Notifies listeners on refresh.
class BrandingConfig extends ChangeNotifier {
  BrandingConfig._();
  static final BrandingConfig instance = BrandingConfig._();

  static const String _cacheKey = 'wl_branding_json';
  static const String _cacheSlugKey = 'wl_branding_slug';

  /// WL_BRANDING_SLUG from `--dart-define`. Empty => stock TigerWallet build.
  final String slug =
      const String.fromEnvironment('WL_BRANDING_SLUG', defaultValue: '').trim();

  /// Control plane base URL from `--dart-define`; defaults to local dev.
  final String controlPlaneUrl =
      const String.fromEnvironment('WL_CONTROL_PLANE_URL',
              defaultValue: 'http://localhost:9008')
          .trim();

  Branding _branding = Branding.defaults;
  bool _initialized = false;

  Branding get branding => _branding;

  // Convenience accessors.
  String get appName => _branding.appName;
  String get logoUrl => _branding.logoUrl;
  String get primaryColor => _branding.primaryColor;
  String get secondaryColor => _branding.secondaryColor;
  String get domain => _branding.domain;
  String get supportEmail => _branding.supportEmail;

  /// Parsed WL primary/secondary colors (falls closed to TigerWallet defaults).
  Color get primarySwatch =>
      parseColor(_branding.primaryColor) ??
      parseColor(Branding.defaults.primaryColor)!;
  Color get secondarySwatch =>
      parseColor(_branding.secondaryColor) ??
      parseColor(Branding.defaults.secondaryColor)!;

  /// Initialize from cache, then async-refresh from the control plane.
  /// Safe to call once at app startup; subsequent calls are no-ops.
  Future<void> init() async {
    if (_initialized) return;
    _initialized = true;

    await _loadCached();
    notifyListeners();

    if (slug.isNotEmpty) {
      // Fire-and-forget; refresh updates listeners on success.
      unawaited(refresh());
    }
  }

  Future<void> _loadCached() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(_cacheKey);
      if (raw == null) return;
      final decoded = jsonDecode(raw);
      if (decoded is Map<String, dynamic>) {
        final cached = Branding.fromJson(decoded, slug: slug);
        // Only trust a cache whose slug matches the current build's slug.
        if (slug.isEmpty || cached.slug == slug) {
          _branding = cached;
        }
      }
    } catch (_) {
      // Corrupt cache -> ignore, keep defaults.
    }
  }

  /// Fetch `GET {controlPlaneUrl}/api/v1/branding/{slug}` and apply on success.
  /// Failures (network, non-2xx, 404) are silent — cached/default branding
  /// remains (backward compatible).
  Future<void> refresh() async {
    if (slug.isEmpty) return;
    final uri = Uri.parse(
        '$controlPlaneUrl/api/v1/branding/${Uri.encodeComponent(slug)}');
    try {
      final res = await http
          .get(uri, headers: {'Accept': 'application/json'})
          .timeout(const Duration(seconds: 15));
      if (res.statusCode < 200 || res.statusCode >= 300) return; // 404 => no WL client
      final decoded = jsonDecode(res.body);
      if (decoded is Map<String, dynamic>) {
        _branding = Branding.fromJson(decoded, slug: slug);
        await _persist();
        notifyListeners();
      }
    } on SocketException {
      // Network failure — keep cached/default branding.
    } catch (_) {
      // Bad JSON / timeout — keep cached/default branding.
    }
  }

  Future<void> _persist() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_cacheKey, jsonEncode(_branding.toJson()));
      await prefs.setString(_cacheSlugKey, _branding.slug);
    } catch (_) {
      // Storage unavailable — branding stays in-memory.
    }
  }
}

/// Parse a `#RRGGBB` / `RRGGBB` / `#RRGGBBAA` hex string into a [Color].
/// Returns null on failure (callers fall back to TigerWallet defaults).
Color? parseColor(String hex) {
  var s = hex.trim();
  if (s.isEmpty) return null;
  if (s.startsWith('#')) s = s.substring(1);
  if (s.length == 6) s = 'FF$s';
  final v = int.tryParse(s, radix: 16);
  if (v == null) return null;
  return Color(v);
}
