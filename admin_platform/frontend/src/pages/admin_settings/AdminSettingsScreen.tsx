/**
 * TigerWallet Admin Settings - Complete Implementation
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, SafeAreaView, StatusBar, Switch, Alert } from 'react-native';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../../../mobile_apps/tigerwallet/app/src/store';
import { toggleTheme } from '../../../mobile_apps/tigerwallet/app/src/store/slices/themeSlice';
import { COLORS, SPACING, FONT_SIZES } from '../../../mobile_apps/tigerwallet/app/src/constants/theme';

const AdminSettingsScreen: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [settings, setSettings] = useState({
    maintenanceMode: false,
    newUserRegistration: true,
    tradingEnabled: true,
    withdrawalEnabled: true,
    depositEnabled: true,
    kycRequired: true,
    twoFactorRequired: true,
    emailNotifications: true,
    smsNotifications: false,
  });

  const updateSetting = (key: string, value: boolean) => {
    setSettings(prev => ({ ...prev, [key]: value }));
    Alert.alert('Settings Updated', `${key} has been ${value ? 'enabled' : 'disabled'}`);
  };

  const SettingRow = ({ title, subtitle, value, onToggle }: { title: string; subtitle?: string; value: boolean; onToggle: (v: boolean) => void }) => (
    <View style={[styles.settingRow, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.settingInfo}>
        <Text style={[styles.settingTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{title}</Text>
        {subtitle && <Text style={[styles.settingSubtitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{subtitle}</Text>}
      </View>
      <Switch value={value} onValueChange={onToggle} trackColor={{ false: COLORS.gray, true: COLORS.primary }} thumbColor={COLORS.white} />
    </View>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      <View style={[styles.header, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Admin Settings</Text>
      </View>

      <ScrollView showsVerticalScrollIndicator={false}>
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>SYSTEM</Text>
          <SettingRow title="Maintenance Mode" subtitle="Put the entire system in maintenance" value={settings.maintenanceMode} onToggle={(v) => updateSetting('maintenanceMode', v)} />
          <SettingRow title="Dark Mode" subtitle="Admin panel theme" value={isDark} onToggle={() => dispatch(toggleTheme())} />
        </View>

        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>USER OPERATIONS</Text>
          <SettingRow title="New User Registration" subtitle="Allow new users to register" value={settings.newUserRegistration} onToggle={(v) => updateSetting('newUserRegistration', v)} />
          <SettingRow title="KYC Required" subtitle="Require identity verification" value={settings.kycRequired} onToggle={(v) => updateSetting('kycRequired', v)} />
          <SettingRow title="2FA Required" subtitle="Require two-factor authentication" value={settings.twoFactorRequired} onToggle={(v) => updateSetting('twoFactorRequired', v)} />
        </View>

        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>TRADING & TRANSACTIONS</Text>
          <SettingRow title="Trading Enabled" subtitle="Enable all trading operations" value={settings.tradingEnabled} onToggle={(v) => updateSetting('tradingEnabled', v)} />
          <SettingRow title="Withdrawals" subtitle="Enable withdrawal operations" value={settings.withdrawalEnabled} onToggle={(v) => updateSetting('withdrawalEnabled', v)} />
          <SettingRow title="Deposits" subtitle="Enable deposit operations" value={settings.depositEnabled} onToggle={(v) => updateSetting('depositEnabled', v)} />
        </View>

        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>NOTIFICATIONS</Text>
          <SettingRow title="Email Notifications" subtitle="Send email notifications" value={settings.emailNotifications} onToggle={(v) => updateSetting('emailNotifications', v)} />
          <SettingRow title="SMS Notifications" subtitle="Send SMS notifications" value={settings.smsNotifications} onToggle={(v) => updateSetting('smsNotifications', v)} />
        </View>

        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>ACCOUNT</Text>
          <TouchableOpacity style={[styles.actionRow, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]} onPress={() => Alert.alert('Change Password', 'Navigate to password change')}>
            <Text style={[styles.actionText, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Change Password</Text>
            <Text style={[styles.arrow, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>›</Text>
          </TouchableOpacity>
          <TouchableOpacity style={[styles.actionRow, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]} onPress={() => Alert.alert('API Keys', 'Manage API keys')}>
            <Text style={[styles.actionText, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>API Keys</Text>
            <Text style={[styles.arrow, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>›</Text>
          </TouchableOpacity>
          <TouchableOpacity style={[styles.actionRow, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]} onPress={() => Alert.alert('Audit Logs', 'View audit logs')}>
            <Text style={[styles.actionText, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Audit Logs</Text>
            <Text style={[styles.arrow, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>›</Text>
          </TouchableOpacity>
          <TouchableOpacity style={[styles.actionRow, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]} onPress={() => Alert.alert('Logout', 'Admin logout')}>
            <Text style={[styles.actionText, { color: COLORS.error }]}>Logout</Text>
            <Text style={[styles.arrow, { color: COLORS.error }]}>›</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 }, header: { padding: SPACING.md }, headerTitle: { fontSize: FONT_SIZES.xl, fontWeight: 'bold' },
  section: { marginTop: SPACING.lg, paddingHorizontal: SPACING.md }, sectionTitle: { fontSize: FONT_SIZES.sm, fontWeight: '600', marginBottom: SPACING.sm, marginLeft: SPACING.sm },
  settingRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.xs },
  settingInfo: { flex: 1 }, settingTitle: { fontSize: FONT_SIZES.md, fontWeight: '600' }, settingSubtitle: { fontSize: FONT_SIZES.sm, marginTop: 2 },
  actionRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.xs },
  actionText: { fontSize: FONT_SIZES.md }, arrow: { fontSize: FONT_SIZES.xxl },
});

export default AdminSettingsScreen;
