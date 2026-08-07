package com.tigeradmin

import android.app.Application
import android.content.Context
import coil.ImageLoader
import coil.ImageLoaderFactory
import coil.disk.DiskCache
import coil.memory.MemoryCache
import coil.request.CachePolicy
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

/**
 * TigerAdmin Application
 * Main application class for the TigerWallet Admin Android App
 * Provides complete admin functionality for platform management
 */
class TigerAdminApplication : Application(), ImageLoaderFactory {

    companion object {
        const val BASE_URL = "https://api.tigerwallet.io/admin/v1/"
        const val WS_URL = "wss://ws.tigerwallet.io/admin"
        
        // Redis configuration
        const val REDIS_HOST = "redis.tigerwallet.io"
        const val REDIS_PORT = 6379
        
        // PostgreSQL configuration
        const val POSTGRES_HOST = "postgres.tigerwallet.io"
        const val POSTGRES_PORT = 5432
        const val POSTGRES_DB = "tigerwallet_admin"
        
        // Cache configuration
        const val CACHE_SIZE_MB = 50L
        const val MEMORY_CACHE_PERCENT = 0.25
        
        // Session configuration
        const val SESSION_TIMEOUT_MINUTES = 30
        const val REFRESH_TOKEN_VALIDITY_DAYS = 30
        
        lateinit var instance: TigerAdminApplication
            private set
    }

    // Network components
    lateinit var okHttpClient: OkHttpClient
        private set
    lateinit var retrofit: Retrofit
        private set
    lateinit var adminApiService: AdminApiService
        private set
    lateinit var webSocketService: AdminWebSocketService
        private set
    
    // Session management
    lateinit var sessionManager: SessionManager
        private set
    
    // Notification service
    lateinit var notificationService: NotificationService
        private set
    
    // Cache manager
    lateinit var cacheManager: CacheManager
        private set
    
    // Analytics
    lateinit var analyticsManager: AnalyticsManager
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this
        initializeNetwork()
        initializeSessionManager()
        initializeNotificationService()
        initializeCacheManager()
        initializeAnalytics()
    }

    private fun initializeNetwork() {
        // OkHttp client with logging, timeouts, and interceptors
        val loggingInterceptor = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        okHttpClient = OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .addInterceptor(loggingInterceptor)
            .addInterceptor { chain ->
                val original = chain.request()
                val requestBuilder = original.newBuilder()
                    .header("Content-Type", "application/json")
                    .header("Accept", "application/json")
                    .header("X-Admin-Platform", "android")
                    .header("X-App-Version", BuildConfig.VERSION_NAME)
                
                // Add auth token if available
                SessionManager.getInstance(this)?.let { session ->
                    session.getAuthToken()?.let { token ->
                        requestBuilder.header("Authorization", "Bearer $token")
                    }
                }
                
                chain.proceed(requestBuilder.build())
            }
            .retryOnConnectionFailure(true)
            .build()

        // Retrofit configuration
        retrofit = Retrofit.Builder()
            .baseUrl(BASE_URL)
            .client(okHttpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .addConverterFactory(EnumConverterFactory())
            .build()

        adminApiService = retrofit.create(AdminApiService::class.java)
        
        // Initialize WebSocket service
        webSocketService = AdminWebSocketService(WS_URL, okHttpClient)
    }

    private fun initializeSessionManager() {
        sessionManager = SessionManager(this)
    }

    private fun initializeNotificationService() {
        notificationService = NotificationService(this)
    }

    private fun initializeCacheManager() {
        cacheManager = CacheManager(this)
    }

    private fun initializeAnalytics() {
        analyticsManager = AnalyticsManager(this)
    }

    override fun newImageLoader(): ImageLoader {
        return ImageLoader.Builder(this)
            .memoryCache {
                MemoryCache.Builder(this)
                    .maxSizePercent(MEMORY_CACHE_PERCENT)
                    .build()
            }
            .diskCache {
                DiskCache.Builder()
                    .directory(cacheDir.resolve("image_cache"))
                    .maxSizeBytes(CACHE_SIZE_MB * 1024 * 1024)
                    .build()
            }
            .memoryCachePolicy(CachePolicy.ENABLED)
            .diskCachePolicy(CachePolicy.ENABLED)
            .crossfade(true)
            .build()
    }

    /**
     * Get the application context
     */
    fun getAppContext(): Context = applicationContext

    /**
     * Check if user is logged in as admin
     */
    fun isLoggedIn(): Boolean = sessionManager.isLoggedIn()

    /**
     * Get current admin user
     */
    fun getCurrentAdmin(): AdminUser? = sessionManager.getCurrentAdmin()

    /**
     * Logout admin user
     */
    fun logout() {
        sessionManager.logout()
        cacheManager.clearAll()
    }

    /**
     * Get API service for making network requests
     */
    fun getApiService(): AdminApiService = adminApiService

    /**
     * Get WebSocket service for real-time updates
     */
    fun getWebSocketService(): AdminWebSocketService = webSocketService
}
