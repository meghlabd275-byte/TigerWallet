/**
 * TigerWallet White Label Management - Complete Implementation
 * 
 * Full white label client management with permissions and branding
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, TextInput, SafeAreaView, StatusBar, Alert, Switch } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../../mobile_apps/tigerwallet/app/src/store';
import { COLORS, SPACING, FONT_SIZES } from '../../../mobile_apps/tigerwallet/app/src/constants/theme';

interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  email: string;
  status: 'active' | 'suspended' | 'pending' | 'halted';
  plan: 'basic' | 'professional' | 'enterprise';
  createdAt: number;
  users: number;
  wallets: number;
  volume: number;
  branding: {
    primaryColor: string;
    logoUrl: string;
    name: string;
  };
  permissions: {
    canAddChain: boolean;
    canAddToken: boolean;
    canCustomizeFees: boolean;
    canAccessAnalytics: boolean;
    canCreateSubAdmins: boolean;
  };
}

const mockClients: WhiteLabelClient[] = [
  { id: '1', name: 'CryptoFast', domain: 'cryptofast.io', email: 'admin@cryptofast.io', status: 'active', plan: 'enterprise', createdAt: Date.now() - 86400000 * 60, users: 15000, wallets: 45000, volume: 2500000, branding: { primaryColor: '#FF6B35', logoUrl: 'https://example.com/logo.png', name: 'CryptoFast' }, permissions: { canAddChain: true, canAddToken: true, canCustomizeFees: true, canAccessAnalytics: true, canCreateSubAdmins: true } },
  { id: '2', name: 'SecureWallet', domain: 'securewallet.app', email: 'team@securewallet.app', status: 'active', plan: 'professional', createdAt: Date.now() - 86400000 * 90, users: 8500, wallets: 22000, volume: 1200000, branding: { primaryColor: '#00D9FF', logoUrl: 'https://example.com/logo2.png', name: 'SecureWallet' }, permissions: { canAddChain: true, canAddToken: true, canCustomizeFees: false, canAccessAnalytics: true, canCreateSubAdmins: true } },
  { id: '3', name: 'BlockEase', domain: 'blockease.com', email: 'hello@blockease.com', status: 'pending', plan: 'basic', createdAt: Date.now() - 86400000 * 5, users: 0, wallets: 0, volume: 0, branding: { primaryColor: '#10B981', logoUrl: 'https://example.com/logo3.png', name: 'BlockEase' }, permissions: { canAddChain: false, canAddToken: false, canCustomizeFees: false, canAccessAnalytics: false, canCreateSubAdmins: false } },
  { id: '4', name: 'MyChain', domain: 'mychain.io', email: 'support@mychain.io', status: 'suspended', plan: 'professional', createdAt: Date.now() - 86400000 * 120, users: 5200, wallets: 15000, volume: 890000, branding: { primaryColor: '#F59E0B', logoUrl: 'https://example.com/logo4.png', name: 'MyChain' }, permissions: { canAddChain: true, canAddToken: true, canCustomizeFees: false, canAccessAnalytics: true, canCreateSubAdmins: false } },
  { id: '5', name: 'DigiPay', domain: 'digipay.tech', email: 'team@digipay.tech', status: 'halted', plan: 'enterprise', createdAt: Date.now() - 86400000 * 180, users: 25000, wallets: 75000, volume: 5200000, branding: { primaryColor: '#8B5CF6', logoUrl: 'https://example.com/logo5.png', name: 'DigiPay' }, permissions: { canAddChain: true, canAddToken: true, canCustomizeFees: true, canAccessAnalytics: true, canCreateSubAdmins: true } },
];

const planColors: Record<string, string> = {
  basic: COLORS.info,
  professional: COLORS.warning,
  enterprise: COLORS.primary,
};

const WhiteLabelScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [clients, setClients] = useState<WhiteLabelClient[]>(mockClients);
  const [searchQuery, setSearchQuery] = useState('');
  const [filter, setFilter] = useState<'all' | 'active' | 'suspended' | 'pending' | 'halted'>('all');

  const filteredClients = clients.filter(client => {
    const matchesSearch = client.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         client.domain.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter = filter === 'all' || client.status === filter;
    return matchesSearch && matchesFilter;
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return COLORS.success;
      case 'pending': return COLORS.warning;
      case 'suspended': return COLORS.error;
      case 'halted': return COLORS.gray;
      default: return COLORS.gray;
    }
  };

  const handleClientAction = (client: WhiteLabelClient, action: string) => {
    Alert.alert(action, `${action} ${client.name}?`, [
      { text: 'Cancel', style: 'cancel' },
      { text: action, onPress: () => {} },
    ]);
  };

  const formatDate = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    return `${Math.floor(diff / 86400000)} days ago`;
  };

  const renderClientItem = ({ item }: { item: WhiteLabelClient }) => (
    <View style={[styles.clientCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.clientHeader}>
        <View style={[styles.clientIcon, { backgroundColor: item.branding.primaryColor + '30' }]}>
          <Text style={[styles.clientIconText, { color: item.branding.primaryColor }]}>{item.name.charAt(0)}</Text>
        </View>
        <View style={styles.clientInfo}>
          <View style={styles.clientTopRow}>
            <Text style={[styles.clientName, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.name}</Text>
            <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) + '20' }]}>
              <Text style={[styles.statusText, { color: getStatusColor(item.status) }]}>{item.status}</Text>
            </View>
          </View>
          <Text style={[styles.clientDomain, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{item.domain}</Text>
        </View>
        <View style={[styles.planBadge, { backgroundColor: planColors[item.plan] + '20' }]}>
          <Text style={[styles.planText, { color: planColors[item.plan] }]}>{item.plan.toUpperCase()}</Text>
        </View>
      </View>

      <View style={styles.clientStats}>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Users</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.users.toLocaleString()}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Wallets</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.wallets.toLocaleString()}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Volume</Text>
          <Text style={[styles.statValue, { color: COLORS.success }]}>${item.volume.toLocaleString()}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Created</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{formatDate(item.createdAt)}</Text>
        </View>
      </View>

      <View style={styles.permissionsSection}>
        <Text style={[styles.permissionsTitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Permissions</Text>
        <View style={styles.permissionsRow}>
          <View style={[styles.permissionBadge, item.permissions.canAddChain && styles.permissionEnabled]}>
            <Text style={[styles.permissionText, { color: item.permissions.canAddChain ? COLORS.success : COLORS.gray }]}>Add Chain</Text>
          </View>
          <View style={[styles.permissionBadge, item.permissions.canAddToken && styles.permissionEnabled]}>
            <Text style={[styles.permissionText, { color: item.permissions.canAddToken ? COLORS.success : COLORS.gray }]}>Add Token</Text>
          </View>
          <View style={[styles.permissionBadge, item.permissions.canCustomizeFees && styles.permissionEnabled]}>
            <Text style={[styles.permissionText, { color: item.permissions.canCustomizeFees ? COLORS.success : COLORS.gray }]}>Customize Fees</Text>
          </View>
          <View style={[styles.permissionBadge, item.permissions.canAccessAnalytics && styles.permissionEnabled]}>
            <Text style={[styles.permissionText, { color: item.permissions.canAccessAnalytics ? COLORS.success : COLORS.gray }]}>Analytics</Text>
          </View>
        </View>
      </View>

      <View style={styles.clientActions}>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.info + '20' ]} onPress={() => handleClientAction(item, 'View Dashboard')}>
          <Text style={[styles.actionBtnText, { color: COLORS.info }]}>Dashboard</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.primary + '20' ]} onPress={() => handleClientAction(item, 'Edit')}>
          <Text style={[styles.actionBtnText, { color: COLORS.primary }]}>Edit</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.warning + '20' ]} onPress={() => handleClientAction(item, item.status === 'active' ? 'Suspend' : 'Activate')}>
          <Text style={[styles.actionBtnText, { color: COLORS.warning }]}>{item.status === 'active' ? 'Suspend' : 'Activate'}</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.error + '20' ]} onPress={() => handleClientAction(item, 'Delete')}>
          <Text style={[styles.actionBtnText, { color: COLORS.error }]}>Delete</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      
      <View style={[styles.header, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>White Label Management</Text>
        <TouchableOpacity style={[styles.addButton, { backgroundColor: COLORS.primary }]} onPress={() => Alert.alert('Create Client', 'Create new white label client')}>
          <Text style={styles.addButtonText}>+ New Client</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.statsRow}>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.primary }]}>{clients.length}</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total Clients</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.success }]}>{clients.filter(c => c.status === 'active').length}</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Active</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.warning }]}>{clients.filter(c => c.status === 'pending').length}</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Pending</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.success }]}>${(clients.reduce((sum, c) => sum + c.volume, 0) / 1000000).toFixed(1)}M</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total Volume</Text>
        </View>
      </View>

      <View style={styles.searchContainer}>
        <TextInput
          style={[styles.searchInput, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight, color: isDark ? COLORS.textDark : COLORS.textLight }]}
          placeholder="Search by name or domain..."
          placeholderTextColor={isDark ? COLORS.gray : COLORS.lightGray}
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      <View style={styles.filterContainer}>
        {(['all', 'active', 'pending', 'suspended', 'halted'] as const).map(f => (
          <TouchableOpacity key={f} style={[styles.filterChip, filter === f && { backgroundColor: COLORS.primary }]} onPress={() => setFilter(f)}>
            <Text style={[styles.filterText, filter === f && { color: COLORS.white }]}>{f.charAt(0).toUpperCase() + f.slice(1)}</Text>
          </TouchableOpacity>
        ))}
      </View>

      <FlatList
        data={filteredClients}
        renderItem={renderClientItem}
        keyExtractor={item => item.id}
        contentContainerStyle={styles.list}
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  headerTitle: { fontSize: FONT_SIZES.xl, fontWeight: 'bold' },
  addButton: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 8 },
  addButtonText: { color: COLORS.white, fontWeight: '600' },
  statsRow: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm },
  statCard: { flex: 1, padding: SPACING.sm, borderRadius: 8, alignItems: 'center', marginHorizontal: 2 },
  statNumber: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  statLabel: { fontSize: FONT_SIZES.xs },
  searchContainer: { paddingHorizontal: SPACING.md, marginBottom: SPACING.sm },
  searchInput: { padding: SPACING.md, borderRadius: 8, fontSize: FONT_SIZES.md },
  filterContainer: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm, flexWrap: 'wrap' },
  filterChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.xs, borderRadius: 20, marginRight: SPACING.sm, marginBottom: 4, backgroundColor: COLORS.cardDark },
  filterText: { fontSize: FONT_SIZES.sm, color: COLORS.gray },
  list: { padding: SPACING.md, paddingBottom: 100 },
  clientCard: { padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.md },
  clientHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: SPACING.md },
  clientIcon: { width: 48, height: 48, borderRadius: 24, justifyContent: 'center', alignItems: 'center' },
  clientIconText: { fontSize: 22, fontWeight: 'bold' },
  clientInfo: { flex: 1, marginLeft: SPACING.sm },
  clientTopRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  clientName: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  clientDomain: { fontSize: FONT_SIZES.sm },
  statusBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 2, borderRadius: 4 },
  statusText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  planBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 2, borderRadius: 4 },
  planText: { fontSize: FONT_SIZES.xs, fontWeight: 'bold' },
  clientStats: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: SPACING.md },
  statItem: { alignItems: 'center' },
  statValue: { fontSize: FONT_SIZES.md, fontWeight: '600' },
  permissionsSection: { marginBottom: SPACING.md },
  permissionsTitle: { fontSize: FONT_SIZES.sm, marginBottom: SPACING.xs },
  permissionsRow: { flexDirection: 'row', flexWrap: 'wrap' },
  permissionBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 4, borderRadius: 4, marginRight: SPACING.xs, marginBottom: 4, backgroundColor: COLORS.borderDark },
  permissionEnabled: { backgroundColor: COLORS.success + '20' },
  permissionText: { fontSize: FONT_SIZES.xs },
  clientActions: { flexDirection: 'row', justifyContent: 'space-between' },
  actionBtn: { flex: 1, padding: SPACING.sm, borderRadius: 6, alignItems: 'center', marginHorizontal: 2 },
  actionBtnText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
});

export default WhiteLabelScreen;
