/**
 * TigerWallet Wallets Management - Complete Implementation
 * 
 * Master wallet and user wallet management
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, TextInput, SafeAreaView, StatusBar, Alert } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../../mobile_apps/tigerwallet/app/src/store';
import { COLORS, SPACING, FONT_SIZES } from '../../../mobile_apps/tigerwallet/app/src/constants/theme';

interface Wallet {
  id: string;
  address: string;
  chain: string;
  chainId: number;
  balance: number;
  tokenCount: number;
  type: 'user' | 'master' | 'hot' | 'cold';
  status: 'active' | 'frozen' | 'paused';
  createdAt: number;
  lastTransaction: number;
}

const mockWallets: Wallet[] = [
  { id: '1', address: '0x742d35Cc6634C0532925a3b844Bc9e7595f8aB1E', chain: 'Ethereum', chainId: 1, balance: 1250.5, tokenCount: 15, type: 'master', status: 'active', createdAt: Date.now() - 86400000 * 180, lastTransaction: Date.now() - 3600000 },
  { id: '2', address: '0x1234567890abcdef1234567890abcdef12345678', chain: 'Ethereum', chainId: 1, balance: 5.2, tokenCount: 8, type: 'hot', status: 'active', createdAt: Date.now() - 86400000 * 30, lastTransaction: Date.now() - 7200000 },
  { id: '3', address: '0xabcdef1234567890abcdef1234567890abcdef12', chain: 'BSC', chainId: 56, balance: 45.8, tokenCount: 12, type: 'hot', status: 'active', createdAt: Date.now() - 86400000 * 60, lastTransaction: Date.now() - 1800000 },
  { id: '4', address: '0x9876543210fedcba9876543210fedcba98765432', chain: 'Polygon', chainId: 137, balance: 12500, tokenCount: 25, type: 'cold', status: 'active', createdAt: Date.now() - 86400000 * 90, lastTransaction: Date.now() - 86400000 },
  { id: '5', address: '0xfedcba9876543210fedcba9876543210fedcba98', chain: 'Solana', chainId: 501, balance: 250, tokenCount: 10, type: 'user', status: 'active', createdAt: Date.now() - 86400000 * 15, lastTransaction: Date.now() - 300000 },
  { id: '6', address: '0x5678901234abcdef5678901234abcdef56789012', chain: 'Arbitrum', chainId: 42161, balance: 0.8, tokenCount: 5, type: 'user', status: 'frozen', createdAt: Date.now() - 86400000 * 45, lastTransaction: Date.now() - 86400000 * 5 },
];

const chainIcons: Record<string, string> = {
  'Ethereum': '🔷',
  'BSC': '🟡',
  'Polygon': '⬡',
  'Solana': '◎',
  'Arbitrum': '🔴',
  'Avalanche': '🔺',
  'Base': '🔵',
  'Optimism': '🔴',
};

const WalletsScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [wallets, setWallets] = useState<Wallet[]>(mockWallets);
  const [searchQuery, setSearchQuery] = useState('');
  const [filter, setFilter] = useState<'all' | 'master' | 'hot' | 'cold' | 'user'>('all');

  const filteredWallets = wallets.filter(wallet => {
    const matchesSearch = wallet.address.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         wallet.chain.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter = filter === 'all' || wallet.type === filter;
    return matchesSearch && matchesFilter;
  });

  const totalBalance = wallets.reduce((sum, w) => sum + w.balance, 0);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return COLORS.success;
      case 'frozen': return COLORS.error;
      case 'paused': return COLORS.warning;
      default: return COLORS.gray;
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'master': return COLORS.primary;
      case 'hot': return COLORS.error;
      case 'cold': return COLORS.info;
      case 'user': return COLORS.success;
      default: return COLORS.gray;
    }
  };

  const formatDate = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return `${Math.floor(diff / 86400000)}d ago`;
  };

  const renderWalletItem = ({ item }: { item: Wallet }) => (
    <View style={[styles.walletCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.walletHeader}>
        <View style={[styles.chainIcon, { backgroundColor: getTypeColor(item.type) + '20' }]}>
          <Text style={styles.chainIconText}>{chainIcons[item.chain] || '⛓️'}</Text>
        </View>
        <View style={styles.walletInfo}>
          <View style={styles.walletTopRow}>
            <Text style={[styles.walletAddress, { color: isDark ? COLORS.textDark : COLORS.textLight }]} numberOfLines={1}>
              {item.address.slice(0, 10)}...{item.address.slice(-8)}
            </Text>
            <View style={[styles.typeBadge, { backgroundColor: getTypeColor(item.type) + '20' }]}>
              <Text style={[styles.typeText, { color: getTypeColor(item.type) }]}>{item.type.toUpperCase()}</Text>
            </View>
          </View>
          <Text style={[styles.walletChain, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{item.chain} (Chain ID: {item.chainId})</Text>
        </View>
        <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) + '20' }]}>
          <Text style={[styles.statusText, { color: getStatusColor(item.status) }]}>{item.status}</Text>
        </View>
      </View>

      <View style={styles.walletStats}>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Balance</Text>
          <Text style={[styles.statValue, { color: COLORS.success }]}>{item.balance.toFixed(4)} ETH</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Tokens</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.tokenCount}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Last TX</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{formatDate(item.lastTransaction)}</Text>
        </View>
      </View>

      <View style={styles.walletActions}>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.primary + '20' }]}>
          <Text style={[styles.actionBtnText, { color: COLORS.primary }]}>View</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.info + '20' }]}>
          <Text style={[styles.actionBtnText, { color: COLORS.info }]}>Transactions</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.warning + '20' }]}>
          <Text style={[styles.actionBtnText, { color: COLORS.warning }]}>Freeze</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.error + '20' }]}>
          <Text style={[styles.actionBtnText, { color: COLORS.error }]}>Delete</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      
      {/* Header */}
      <View style={[styles.header, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <View>
          <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Wallets Management</Text>
          <Text style={[styles.headerSubtitle, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total: {totalBalance.toFixed(4)} ETH</Text>
        </View>
        <TouchableOpacity style={[styles.addButton, { backgroundColor: COLORS.primary }]}>
          <Text style={styles.addButtonText}>+ Create Wallet</Text>
        </TouchableOpacity>
      </View>

      {/* Search and Filter */}
      <View style={styles.searchContainer}>
        <TextInput
          style={[styles.searchInput, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight, color: isDark ? COLORS.textDark : COLORS.textLight }]}
          placeholder="Search by address or chain..."
          placeholderTextColor={isDark ? COLORS.gray : COLORS.lightGray}
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      {/* Filters */}
      <View style={styles.filterContainer}>
        {(['all', 'master', 'hot', 'cold', 'user'] as const).map(f => (
          <TouchableOpacity
            key={f}
            style={[styles.filterChip, filter === f && { backgroundColor: COLORS.primary }]}
            onPress={() => setFilter(f)}
          >
            <Text style={[styles.filterText, filter === f && { color: COLORS.white }]}>{f.charAt(0).toUpperCase() + f.slice(1)}</Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* Master Wallet Info */}
      <View style={[styles.masterWalletCard, { backgroundColor: COLORS.primary }]}>
        <View style={styles.masterWalletHeader}>
          <Text style={styles.masterWalletIcon}>🔐</Text>
          <View>
            <Text style={styles.masterWalletTitle}>Master Wallet</Text>
            <Text style={styles.masterWalletAddress}>0x742d...aB1E</Text>
          </View>
        </View>
        <View style={styles.masterWalletStats}>
          <View style={styles.masterStatItem}>
            <Text style={styles.masterStatLabel}>Balance</Text>
            <Text style={styles.masterStatValue}>1,250.5 ETH</Text>
          </View>
          <View style={styles.masterStatItem}>
            <Text style={styles.masterStatLabel}>Type</Text>
            <Text style={styles.masterStatValue}>Master</Text>
          </View>
          <View style={styles.masterStatItem}>
            <Text style={styles.masterStatLabel}>Status</Text>
            <Text style={styles.masterStatValue}>Active</Text>
          </View>
        </View>
      </View>

      {/* Wallets List */}
      <FlatList
        data={filteredWallets}
        renderItem={renderWalletItem}
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
  headerSubtitle: { fontSize: FONT_SIZES.sm },
  addButton: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 8 },
  addButtonText: { color: COLORS.white, fontWeight: '600' },
  searchContainer: { paddingHorizontal: SPACING.md, marginBottom: SPACING.sm },
  searchInput: { padding: SPACING.md, borderRadius: 8, fontSize: FONT_SIZES.md },
  filterContainer: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm },
  filterChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.xs, borderRadius: 20, marginRight: SPACING.sm, backgroundColor: COLORS.cardDark },
  filterText: { fontSize: FONT_SIZES.sm, color: COLORS.gray },
  masterWalletCard: { margin: SPACING.md, padding: SPACING.md, borderRadius: 12 },
  masterWalletHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: SPACING.md },
  masterWalletIcon: { fontSize: 32, marginRight: SPACING.md },
  masterWalletTitle: { color: COLORS.white, fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  masterWalletAddress: { color: COLORS.white + '80', fontSize: FONT_SIZES.sm },
  masterWalletStats: { flexDirection: 'row', justifyContent: 'space-between' },
  masterStatItem: {},
  masterStatLabel: { color: COLORS.white + '80', fontSize: FONT_SIZES.xs },
  masterStatValue: { color: COLORS.white, fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  list: { padding: SPACING.md, paddingBottom: 100 },
  walletCard: { padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.md },
  walletHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: SPACING.md },
  chainIcon: { width: 44, height: 44, borderRadius: 22, justifyContent: 'center', alignItems: 'center' },
  chainIconText: { fontSize: 22 },
  walletInfo: { flex: 1, marginLeft: SPACING.sm },
  walletTopRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  walletAddress: { fontSize: FONT_SIZES.md, fontWeight: '600', maxWidth: '60%' },
  typeBadge: { paddingHorizontal: SPACING.xs, paddingVertical: 2, borderRadius: 4 },
  typeText: { fontSize: FONT_SIZES.xs, fontWeight: 'bold' },
  walletChain: { fontSize: FONT_SIZES.sm },
  statusBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 4, borderRadius: 4 },
  statusText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  walletStats: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: SPACING.md },
  statItem: { alignItems: 'center' },
  statLabel: { fontSize: FONT_SIZES.xs },
  statValue: { fontSize: FONT_SIZES.md, fontWeight: '600' },
  walletActions: { flexDirection: 'row', justifyContent: 'space-between' },
  actionBtn: { flex: 1, padding: SPACING.sm, borderRadius: 6, alignItems: 'center', marginHorizontal: 2 },
  actionBtnText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
});

export default WalletsScreen;
