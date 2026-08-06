/**
 * TigerWallet Admin - Navigation
 */

package com.tigerwallet.admin.ui.navigation

import androidx.compose.runtime.Composable
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.tigerwallet.admin.ui.screens.auth.LoginScreen
import com.tigerwallet.admin.ui.screens.auth.SplashScreen
import com.tigerwallet.admin.ui.screens.dashboard.DashboardScreen
import com.tigerwallet.admin.ui.screens.users.UsersScreen
import com.tigerwallet.admin.ui.screens.kyc.KycScreen
import com.tigerwallet.admin.ui.screens.transactions.TransactionsScreen
import com.tigerwallet.admin.ui.screens.tokens.TokensScreen
import com.tigerwallet.admin.ui.screens.pairs.PairsScreen
import com.tigerwallet.admin.ui.screens.blockchains.BlockchainsScreen
import com.tigerwallet.admin.ui.screens.whitelabels.WhiteLabelsScreen
import com.tigerwallet.admin.ui.screens.tickets.TicketsScreen
import com.tigerwallet.admin.ui.screens.analytics.AnalyticsScreen
import com.tigerwallet.admin.ui.screens.settings.SettingsScreen

sealed class Screen(val route: String) {
    object Splash : Screen("splash")
    object Login : Screen("login")
    object Dashboard : Screen("dashboard")
    object Users : Screen("users")
    object UserDetail : Screen("users/{userId}") {
        fun createRoute(userId: String) = "users/$userId"
    }
    object Kyc : Screen("kyc")
    object Transactions : Screen("transactions")
    object TransactionDetail : Screen("transactions/{txId}") {
        fun createRoute(txId: String) = "transactions/$txId"
    }
    object Tokens : Screen("tokens")
    object Pairs : Screen("pairs")
    object Blockchains : Screen("blockchains")
    object WhiteLabels : Screen("whitelabels")
    object Tickets : Screen("tickets")
    object Analytics : Screen("analytics")
    object Settings : Screen("settings")
    object Profile : Screen("settings/profile")
}

@Composable
fun TigerAdminNavHost(
    navController: NavHostController = rememberNavController()
) {
    NavHost(
        navController = navController,
        startDestination = Screen.Splash.route
    ) {
        composable(Screen.Splash.route) {
            SplashScreen(
                onNavigateToLogin = {
                    navController.navigate(Screen.Login.route) {
                        popUpTo(Screen.Splash.route) { inclusive = true }
                    }
                }
            )
        }

        composable(Screen.Login.route) {
            LoginScreen(
                onLoginSuccess = {
                    navController.navigate(Screen.Dashboard.route) {
                        popUpTo(Screen.Login.route) { inclusive = true }
                    }
                }
            )
        }

        composable(Screen.Dashboard.route) {
            DashboardScreen(
                onNavigateToUsers = { navController.navigate(Screen.Users.route) },
                onNavigateToKyc = { navController.navigate(Screen.Kyc.route) },
                onNavigateToTransactions = { navController.navigate(Screen.Transactions.route) },
                onNavigateToSettings = { navController.navigate(Screen.Settings.route) }
            )
        }

        composable(Screen.Users.route) {
            UsersScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToUserDetail = { userId ->
                    navController.navigate(Screen.UserDetail.createRoute(userId))
                }
            )
        }

        composable(
            route = Screen.UserDetail.route,
            arguments = listOf(navArgument("userId") { type = NavType.StringType })
        ) { backStackEntry ->
            val userId = backStackEntry.arguments?.getString("userId") ?: ""
            com.tigerwallet.admin.ui.screens.users.UserDetailScreen(
                userId = userId,
                onNavigateBack = { navController.popBackStack() }
            )
        }

        composable(Screen.Kyc.route) {
            KycScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.Transactions.route) {
            TransactionsScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToDetail = { txId ->
                    navController.navigate(Screen.TransactionDetail.createRoute(txId))
                }
            )
        }

        composable(Screen.Tokens.route) {
            TokensScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.Pairs.route) {
            PairsScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.Blockchains.route) {
            BlockchainsScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.WhiteLabels.route) {
            WhiteLabelsScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.Tickets.route) {
            TicketsScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.Analytics.route) {
            AnalyticsScreen(onNavigateBack = { navController.popBackStack() })
        }

        composable(Screen.Settings.route) {
            SettingsScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToProfile = { navController.navigate(Screen.Profile.route) }
            )
        }

        composable(Screen.Profile.route) {
            com.tigerwallet.admin.ui.screens.settings.ProfileScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }
    }
}
