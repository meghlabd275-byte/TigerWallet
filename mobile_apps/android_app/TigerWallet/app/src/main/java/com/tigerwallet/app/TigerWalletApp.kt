package com.tigerwallet.app

import android.app.Application
import com.tigerwallet.app.data.services.ServiceLocator

class TigerWalletApp : Application() {
    
    override fun onCreate() {
        super.onCreate()
        instance = this
        ServiceLocator.init()
    }
    
    companion object {
        lateinit var instance: TigerWalletApp
            private set
    }
}
