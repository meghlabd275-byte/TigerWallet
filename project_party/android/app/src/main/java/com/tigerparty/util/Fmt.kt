package com.tigerparty.util

import android.view.View
import android.widget.TextView
import org.json.JSONArray
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

/**
 * Small shared helpers for rendering backend payloads.
 *
 * Every helper tolerates a missing or differently-typed field by returning a
 * neutral placeholder. That is deliberate: the alternative (throwing) would
 * blank an entire screen because one optional column was absent, and the
 * alternative of inventing a value would be worse.
 */
object Fmt {

    fun text(v: TextView, value: String?) {
        v.text = if (value.isNullOrBlank()) "—" else value
    }

    fun setVisible(v: View, visible: Boolean) {
        v.visibility = if (visible) View.VISIBLE else View.GONE
    }

    /** Reads a string field, accepting a numeric or boolean JSON value too. */
    fun str(o: JSONObject?, key: String): String {
        if (o == null || !o.has(key) || o.isNull(key)) return ""
        return when (val v = o.get(key)) {
            is String -> v
            is Number, is Boolean -> v.toString()
            else -> ""
        }
    }

    fun num(o: JSONObject?, key: String): Double {
        if (o == null || !o.has(key) || o.isNull(key)) return 0.0
        return when (val v = o.get(key)) {
            is Number -> v.toDouble()
            is String -> v.toDoubleOrNull() ?: 0.0
            else -> 0.0
        }
    }

    /** A backend list may arrive as a bare array or wrapped in a named key. */
    fun array(o: JSONObject?, vararg keys: String): JSONArray {
        if (o == null) return JSONArray()
        for (k in keys) {
            val v = o.opt(k)
            if (v is JSONArray) return v
        }
        return JSONArray()
    }

    /**
     * Renders an epoch value as a local timestamp. Accepts seconds or
     * milliseconds; anything that is not a plausible epoch returns "".
     */
    fun epoch(seconds: Long?): String {
        if (seconds == null || seconds <= 0) return ""
        val millis = if (seconds < 100_000_000_000L) seconds * 1000 else seconds
        val fmt = SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.US)
        fmt.timeZone = TimeZone.getDefault()
        return fmt.format(Date(millis))
    }

    fun time(o: JSONObject?, vararg keys: String): String {
        for (k in keys) {
            val raw = str(o, k)
            if (raw.isBlank()) continue
            raw.toLongOrNull()?.let { return epoch(it) }
            // Already a formatted timestamp.
            if (raw.length >= 10) return raw
        }
        return ""
    }

    /** Human status label: lower-cased and trimmed, never invented. */
    fun status(o: JSONObject?): String = str(o, "status").lowercase(Locale.US)

    fun pct(value: Double): String = String.format(Locale.US, "%.2f%%", value)

    fun money(value: Double): String = String.format(Locale.US, "%,.2f", value)
}
