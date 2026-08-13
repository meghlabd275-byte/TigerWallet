/**
 * TigerWallet History Screen - Complete Implementation
 *
 * Transaction history with filters. Data fetched live from the canonical
 * backend (go/wallet_api) via APIService.getTransactions. No mock data.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, SafeAreaView, StatusBar, ActivityIndicator } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';
import { API } from '../../services/API';

interface HistoryTx {
  id: string;
  type: 'send' | 'receive' | 'swap' | 'stake' | 'approve';
  symbol: string;
  amount: string;
  status: 'pending' | 'completed' | 'failed';
  timestamp: number;
  hash: string;
  from?: string;
  to?: string;
}

const HistoryScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const walletState = useSelector((state: RootState) => state.wallet);
  const isDark = theme === 'dark';
  const [filter, setFilter] = useState<'all' | 'send' | 'receive' | 'swap'>('all');
  const [transactions, setTransactions] = useState<HistoryTx[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTransactions = useCallback(async () => {
    const wallet = walletState.wallet;
    if (!wallet || !wallet.id) {
      setTransactions([]);
      setLoading(false);
      return;
    }
    const chainId = walletState.selectedChainId || 1;
    const address = wallet.addresses?.[chainId];
    setLoading(true);
    setError(null);
    try {
      const res = await API.getTransactions(wallet.id, chainId);
      const raw: any[] = res?.data?.transactions ?? [];
      const mapped: HistoryTx[] = raw.map((t: any) => {
        const isSend = address && t.from && String(t.from).toLowerCase() === String(address).toLowerCase();
        return {
          id: String(t.hash ?? t.id ?? Math.random().toString(36)),
          type: isSend ? 'send' : 'receive',
          symbol: t.token_symbol || t.symbol || 'ETH',
          amount: t.value ? String(t.value) : '0',
          status: t.status === 'success' || t.status === 'confirmed' ? 'completed' : (t.isError === '1' ? 'failed' : 'pending'),
          timestamp: Number(t.timestamp || t.timeStamp || 0) * (String(t.timestamp || t.timeStamp || '').length > 10 ? 1 : 1000),
          hash: String(t.hash ?? ''),
          from: t.from ? String(t.from) : undefined,
          to: t.to ? String(t.to) : undefined,
        };
      });
      setTransactions(mapped);
    } catch (e: any) {
      setError(e?.message || 'Failed to load transactions');
      setTransactions([]);
    } finally {
      setLoading(false);
    }
  }, [walletState.wallet, walletState.selectedChainId]);

  useEffect(() => {
    loadTransactions();
  }, [loadTransactions]);

  const filteredTx = filter === 'all' ? transactions : transactions.filter(t => t.type === filter);

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'send': return '>>>';
      case 'receive': return '<<<';
      case 'swap': return '><>';
      case 'stake': return '<>';
      case 'approve': return '[+]';
      default: return '[]';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return COLORS.success;
      case 'pending': return COLORS.warning;
      case 'failed': return COLORS.error;
      default: return COLORS.gray;
    }
  };

  const formatTime = (timestamp: number) => {
    if (!timestamp) return '';
    const diff = Date.now() - timestamp;
    if (diff < 0) return 'just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return `${Math.floor(diff / 86400000)}d ago`;
  };

  const renderTx = ({ item }: { item: HistoryTx }) => (
    <View style={[styles.txItem, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.txIconContainer}>
        <Text style={styles.txIcon}>{getTypeIcon(item.type)}</Text>
      </View>
      <View style={styles.txInfo}>
        <Text style={[styles.txTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          {item.type.charAt(0).toUpperCase() + item.type.slice(1)} {item.symbol}
        </Text>
        <Text style={[styles.txHash, { color: isDark ? COLORS.gray : COLORS.lightGray }]} numberOfLines={1}>{item.hash}</Text>
      </View>
      <View style={styles.txRight}>
        <Text style={[styles.txAmount, { color: item.type === 'send' ? COLORS.error : COLORS.success }]}>
          {item.type === 'send' ? '-' : '+'}{item.amount}
        </Text>
        <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) + '20' }]}>
          <Text style={[styles.statusText, { color: getStatusColor(item.status) }]}>{item.status}</Text>
        </View>
        <Text style={[styles.txTime, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{formatTime(item.timestamp)}</Text>
      </View>
    </View>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      <View style={styles.header}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>History</Text>
        <ThemeToggle />
      </View>
      <View style={styles.filterContainer}>
        {(['all', 'send', 'receive', 'swap'] as const).map(f => (
          <TouchableOpacity key={f} style={[styles.filterChip, filter === f && styles.filterChipSelected]} onPress={() => setFilter(f)}>
            <Text style={[styles.filterChipText, filter === f && styles.filterChipTextSelected]}>{f.charAt(0).toUpperCase() + f.slice(1)}</Text>
          </TouchableOpacity>
        ))}
      </View>
      {loading ? (
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color={COLORS.primary} />
        </View>
      ) : error ? (
        <View style={styles.emptyContainer}>
          <Text style={[styles.emptyText, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{error}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={loadTransactions}>
            <Text style={styles.retryText}>Retry</Text>
          </TouchableOpacity>
        </View>
      ) : filteredTx.length === 0 ? (
        <View style={styles.emptyContainer}>
          <Text style={[styles.emptyText, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>No transactions yet</Text>
        </View>
      ) : (
        <FlatList data={filteredTx} renderItem={renderTx} keyExtractor={item => item.id} contentContainerStyle={styles.txList} showsVerticalScrollIndicator={false} />
      )}
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  headerTitle: { fontSize: FONT_SIZES.xxl, fontWeight: 'bold' },
  filterContainer: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm },
  filterChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 20, marginRight: SPACING.sm, backgroundColor: COLORS.cardDark },
  filterChipSelected: { backgroundColor: COLORS.primary },
  filterChipText: { fontSize: FONT_SIZES.sm, fontWeight: '600', color: COLORS.gray },
  filterChipTextSelected: { color: COLORS.white },
  txList: { paddingHorizontal: SPACING.md, paddingBottom: SPACING.xl },
  txItem: { flexDirection: 'row', alignItems: 'center', padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.sm },
  txIconContainer: { width: 44, height: 44, borderRadius: 22, backgroundColor: COLORS.primary + '20', justifyContent: 'center', alignItems: 'center' },
  txIcon: { fontSize: 16, fontWeight: 'bold' },
  txInfo: { flex: 1, marginLeft: SPACING.sm },
  txTitle: { fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  txHash: { fontSize: FONT_SIZES.xs, marginTop: 2 },
  txRight: { alignItems: 'flex-end' },
  txAmount: { fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  statusBadge: { paddingHorizontal: SPACING.xs, paddingVertical: 2, borderRadius: 4, marginTop: 2 },
  statusText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  txTime: { fontSize: FONT_SIZES.xs, marginTop: 2 },
  loadingContainer: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  emptyContainer: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: SPACING.xl },
  emptyText: { fontSize: FONT_SIZES.md, textAlign: 'center' },
  retryButton: { marginTop: SPACING.md, paddingHorizontal: SPACING.lg, paddingVertical: SPACING.sm, borderRadius: 8, backgroundColor: COLORS.primary },
  retryText: { color: COLORS.white, fontWeight: '600' },
});

export default HistoryScreen;
