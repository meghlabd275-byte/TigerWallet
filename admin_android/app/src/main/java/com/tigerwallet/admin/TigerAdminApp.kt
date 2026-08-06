/**
 * TigerWallet Admin - Android Application
 * Complete Native Implementation with Dark/Light Theme Support
 */

package com.tigerwallet.admin

import android.app.Application
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class TigerAdminApp : Application() {

    override fun onCreate() {
        super.onCreate()
        // Initialize app components
    }
}
