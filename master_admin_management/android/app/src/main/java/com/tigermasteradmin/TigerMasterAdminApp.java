package com.tigermasteradmin;

import android.app.Application;
import android.content.SharedPreferences;
import androidx.preference.PreferenceManager;

public class TigerMasterAdminApp extends Application {
    private static TigerMasterAdminApp instance;
    private SharedPreferences preferences;
    
    @Override
    public void onCreate() {
        super.onCreate();
        instance = this;
        preferences = PreferenceManager.getDefaultSharedPreferences(this);
    }
    
    public static TigerMasterAdminApp getInstance() {
        return instance;
    }
    
    public SharedPreferences getPreferences() {
        return preferences;
    }
    
    public String getBaseURL() {
        return "http://localhost:9091";
    }
    
    public boolean isDarkMode() {
        return preferences.getBoolean("dark_mode", false);
    }
    
    public void setDarkMode(boolean enabled) {
        preferences.edit().putBoolean("dark_mode", enabled).apply();
    }
}
