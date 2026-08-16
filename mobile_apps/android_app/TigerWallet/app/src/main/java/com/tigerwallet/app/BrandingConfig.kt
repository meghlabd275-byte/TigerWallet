package com.tigerwallet.app

import android.content.Context
import android.graphics.Color
import androidx.compose.ui.graphics.Color as ComposeColor
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * White-label branding config for the Android app.
 *
 * Loading order:
 *  1. `BuildConfig.WL_BRANDING_SLUG` — injected at build time via the gradle
 *     `buildConfigField` (per WL-client build). Empty/absent => this is a
 *     stock TigerWallet build; no remote fetch happens.
 *  2. If a slug is set, fetch `GET {CONTROL_PLANE_URL}/api/v1/branding/{slug}`
 *     on app startup (OkHttp, already a dep). The endpoint is PUBLIC (no auth)
 *     so a WL-branded app needs no secrets.
 *  3. Cache the fetched config in SharedPreferences so a transient network
 *     failure or cold start still shows the WL brand instead of TigerWallet.
 *  4. Fall back to TigerWallet defaults if there is no slug, the fetch fails,
 *     or the endpoint returns 404 (no WL client matches the slug).
 *
 * `strings.xml` `app_name` remains "TigerWallet" as the launcher label default;
 * [appName] overrides the in-app displayed name at runtime (the launcher label
 * can't be changed at runtime without an activity-alias, which is out of scope
 * for this layer).
 */
object BrandingConfig {

    private const val PREFS_NAME = "wl_branding_prefs"
    private const val KEY_CACHE = "branding_json"
    private const val KEY_SLUG = "branding_slug"

    // CONTROL_PLANE_URL: env-injected at build time via gradle buildConfigField
    // (WL_CONTROL_PLANE_URL). Falls back to the local dev control plane when
    // unset (stock TigerWallet dev build).
    private val controlPlaneUrl: String =
        com.tigerwallet.app.BuildConfig.WL_CONTROL_PLANE_URL
            .takeIf { it.isNotBlank() }
            ?: "http://localhost:9008"

    // TigerWallet stock branding — the backward-compatible default.
    val DEFAULTS = Branding(
        appName = "TigerWallet",
        logoUrl = "",
        primaryColor = "#FF6B35",
        secondaryColor = "#1E3A5F",
        domain = "tigerwallet.io",
        supportEmail = "support@tigerwallet.io",
        termsUrl = "https://tigerwallet.io/terms",
        privacyUrl = "https://tigerwallet.io/privacy",
    )

    private val _current = MutableStateFlow(DEFAULTS)
    val current: StateFlow<Branding> = _current.asStateFlow()

    // WL_BRANDING_SLUG: env-injected at build time via gradle buildConfigField.
    // Empty => stock TigerWallet build; no remote fetch happens.
    val slug: String = com.tigerwallet.app.BuildConfig.WL_BRANDING_SLUG.trim()

    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .build()

    /**
     * Load branding synchronously from the cache, then kick off an async
     * refresh from the control plane. Call from [TigerWalletApp.onCreate] /
     * [ServiceLocator.init].
     */
    fun init(context: Context) {
        loadFromCache(context)
        if (slug.isNotBlank()) {
            refreshAsync(context)
        }
    }

    private fun loadFromCache(context: Context) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val cachedSlug = prefs.getString(KEY_SLUG, null)
        val cachedJson = prefs.getString(KEY_CACHE, null)
        // Only trust the cache if it matches the current build's slug.
        if (cachedSlug == slug && cachedJson != null) {
            _current.value = parse(cachedJson) ?: DEFAULTS
        } else if (cachedJson != null && slug.isBlank() && cachedSlug.isNullOrBlank()) {
            _current.value = parse(cachedJson) ?: DEFAULTS
        } else {
            _current.value = DEFAULTS
        }
    }

    private fun refreshAsync(context: Context) {
        // Fire-and-forget on the IO dispatcher; never block app startup.
        kotlinx.coroutines.GlobalScope.launch(Dispatchers.IO) {
            try {
                val req = Request.Builder()
                    .url("$controlPlaneUrl/api/v1/branding/${slug.trim()}")
                    .get()
                    .build()
                client.newCall(req).execute().use { resp ->
                    if (!resp.isSuccessful) return@use
                    val body = resp.body?.string().orEmpty()
                    val parsed = parse(body) ?: return@use
                    _current.value = parsed
                    persist(context, body)
                }
            } catch (_: Exception) {
                // Network failure: keep the cached/default branding. Never crash.
            }
        }
    }

    private fun persist(context: Context, json: String) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit()
            .putString(KEY_SLUG, slug)
            .putString(KEY_CACHE, json)
            .apply()
    }

    private fun parse(json: String): Branding? {
        return try {
            val o = JSONObject(json)
            // Merge over defaults so a partial WL config still has sensible
            // TigerWallet fallbacks for any missing field.
            DEFAULTS.copy(
                appName = o.optString("app_name", DEFAULTS.appName),
                logoUrl = o.optString("logo_url", DEFAULTS.logoUrl),
                primaryColor = o.optString("primary_color", DEFAULTS.primaryColor),
                secondaryColor = o.optString("secondary_color", DEFAULTS.secondaryColor),
                domain = o.optString("domain", DEFAULTS.domain),
                supportEmail = o.optString("support_email", DEFAULTS.supportEmail),
                termsUrl = o.optString("terms_url", DEFAULTS.termsUrl),
                privacyUrl = o.optString("privacy_url", DEFAULTS.privacyUrl),
            )
        } catch (_: Exception) {
            null
        }
    }

    // --- Convenience accessors ---

    val appName: String get() = _current.value.appName
    val logoUrl: String get() = _current.value.logoUrl
    val primaryColor: String get() = _current.value.primaryColor
    val secondaryColor: String get() = _current.value.secondaryColor
    val domain: String get() = _current.value.domain
    val supportEmail: String get() = _current.value.supportEmail

    /** Parse the WL primary color into an Android color int (fails closed to the default). */
    fun primaryColorInt(): Int = toColorInt(primaryColor, DEFAULTS.primaryColor)
    fun secondaryColorInt(): Int = toColorInt(secondaryColor, DEFAULTS.secondaryColor)

    fun primaryComposeColor(): ComposeColor = ComposeColor(primaryColorInt())
    fun secondaryComposeColor(): ComposeColor = ComposeColor(secondaryColorInt())

    private fun toColorInt(hex: String, fallback: String): Int {
        return try {
            Color.parseColor(normalizeHex(hex))
        } catch (_: Exception) {
            try { Color.parseColor(fallback) } catch (_: Exception) { Color.BLACK }
        }
    }

    private fun normalizeHex(hex: String): String {
        val h = hex.trim()
        return if (h.startsWith("#")) h else "#$h"
    }
}

data class Branding(
    val appName: String,
    val logoUrl: String,
    val primaryColor: String,
    val secondaryColor: String,
    val domain: String,
    val supportEmail: String,
    val termsUrl: String,
    val privacyUrl: String,
)
