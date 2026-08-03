package com.tigerwallet.admin.data.api

import android.content.Context
import com.tigerwallet.admin.util.PreferencesManager
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

object NetworkClient {
    private const val BASE_URL = "https://admin-api.tigerwallet.com/"
    private const val CACHE_SIZE = 10 * 1024 * 1024L // 10 MB

    private var retrofit: Retrofit? = null
    private var apiService: ApiService? = null
    private lateinit var preferencesManager: PreferencesManager

    fun initialize(context: Context) {
        preferencesManager = PreferencesManager(context)
    }

    private val authInterceptor = Interceptor { chain ->
        val originalRequest = chain.request()
        val token = preferencesManager.getToken()

        val newRequest = if (token != null) {
            originalRequest.newBuilder()
                .header("Authorization", "Bearer $token")
                .build()
        } else {
            originalRequest
        }

        chain.proceed(newRequest)
    }

    private val loggingInterceptor = HttpLoggingInterceptor().apply {
        level = HttpLoggingInterceptor.Level.BODY
    }

    private val okHttpClient = OkHttpClient.Builder()
        .addInterceptor(authInterceptor)
        .addInterceptor(loggingInterceptor)
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    fun getApiService(): ApiService {
        if (apiService == null) {
            retrofit = Retrofit.Builder()
                .baseUrl(BASE_URL)
                .client(okHttpClient)
                .addConverterFactory(GsonConverterFactory.create())
                .build()

            apiService = retrofit!!.create(ApiService::class.java)
        }
        return apiService!!
    }

    fun clearToken() {
        preferencesManager.clearToken()
    }
}
