package com.tigerwallet.app

import android.app.Application
import com.tigerwallet.app.data.services.ServiceLocator

class TigerWalletApp : Application() {
    
    override fun onCreate() {
        super.onCreate()
        instance = this
        // Load WL branding (from cache, then async-refresh from the control
        // plane). Stock TigerWallet builds fall back to defaults when no slug
        // is injected at build time.
        BrandingConfig.init(this)
        ServiceLocator.init()
    }
    
    companion object {
        lateinit var instance: TigerWalletApp
            private set
    }
}
