/**
 * TigerWallet Admin - Main Activity
 * Complete Dark/Light Theme Support
 */

package com.tigerwallet.admin.ui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.tigerwallet.admin.ui.navigation.TigerAdminNavHost
import com.tigerwallet.admin.ui.theme.TigerAdminTheme
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        setContent {
            var isDarkTheme by remember { mutableStateOf<Boolean?>(null) }
            val systemDark = isSystemInDarkTheme()

            TigerAdminTheme(
                darkTheme = isDarkTheme ?: systemDark,
                content = {
                    Surface(
                        modifier = Modifier.fillMaxSize(),
                        color = MaterialTheme.colorScheme.background
                    ) {
                        TigerAdminNavHost()
                    }
                }
            )
        }
    }
}
