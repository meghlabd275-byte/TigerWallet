package com.tigerparty

import android.app.Application
import android.content.Context
import androidx.appcompat.app.AppCompatDelegate

/**
 * Application entry point.
 *
 * Theme mode is persisted here (it is not sensitive) and applied before any
 * activity is created, so the window never flashes the wrong palette.
 */
class TigerPartyApp : Application() {

    override fun onCreate() {
        super.onCreate()
        AppCompatDelegate.setDefaultNightMode(readNightMode(this))
    }

    companion object {
        private const val PREFS = "tigerparty_settings"
        private const val KEY_NIGHT_MODE = "night_mode"

        const val MODE_SYSTEM = 0
        const val MODE_LIGHT = 1
        const val MODE_DARK = 2

        fun readNightMode(context: Context): Int {
            val stored = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .getInt(KEY_NIGHT_MODE, MODE_SYSTEM)
            return when (stored) {
                MODE_LIGHT -> AppCompatDelegate.MODE_NIGHT_NO
                MODE_DARK -> AppCompatDelegate.MODE_NIGHT_YES
                else -> AppCompatDelegate.MODE_NIGHT_FOLLOW_SYSTEM
            }
        }

        fun writeNightMode(context: Context, mode: Int) {
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .edit().putInt(KEY_NIGHT_MODE, mode).apply()
            AppCompatDelegate.setDefaultNightMode(readNightMode(context))
        }

        fun currentMode(context: Context): Int =
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .getInt(KEY_NIGHT_MODE, MODE_SYSTEM)
    }
}
