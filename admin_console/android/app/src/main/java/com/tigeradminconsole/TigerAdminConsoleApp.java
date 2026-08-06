package com.tigeradminconsole;

import android.app.Application;
import android.content.SharedPreferences;
import androidx.preference.PreferenceManager;

public class TigerAdminConsoleApp extends Application {
    private static TigerAdminConsoleApp instance;
    private SharedPreferences preferences;
    
    @Override
    public void onCreate() {
        super.onCreate();
        instance = this;
        preferences = PreferenceManager.getDefaultSharedPreferences(this);
    }
    
    public static TigerAdminConsoleApp getInstance() { return instance; }
    public SharedPreferences getPreferences() { return preferences; }
    public String getBaseURL() { return "http://localhost:9093"; }
    public boolean isDarkMode() { return preferences.getBoolean("dark_mode", false); }
    public void setDarkMode(boolean enabled) { preferences.edit().putBoolean("dark_mode", enabled).apply(); }
}
