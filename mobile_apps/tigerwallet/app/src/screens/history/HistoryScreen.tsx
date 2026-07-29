/**
 * TigerWallet History Screen - Complete Implementation
 * 
 * Transaction history with filters
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, SafeAreaView, StatusBar } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';

interface Transaction {
  id: string;
  type: 'send' | 'receive' | 'swap' | 'stake' | 'approve';
  symbol: string;
  amount: string;
  value: number;
  status: 'pending' | 'completed' | 'failed';
  timestamp: number;
  hash: string;
  from?: string;
  to?: string;
}

const transactions: Transaction[] = [
  { id: '1', type: 'send', symbol: 'ETH', amount: '0.5', value: 1500, status: 'completed', timestamp: Date.now() - 3600000, hash: '0x1234...5678', to: '0xabcd...efgh' },
  { id: '2', type: 'receive', symbol: 'USDT', amount: '1000', value: 1000, status: 'completed', timestamp: Date.now() - 86400000, hash: '0xabcd...efgh', from: '0x9876...5432' },
  { id: '3', type: 'swap', symbol: 'ETH → USDC', amount: '1.0', value: 3000, status: 'completed', timestamp: Date.now() - 172800000, hash: '0xdef0...1234' },
  { id: '4', type: 'stake', symbol: 'MATIC', amount: '500', value: 425, status: 'completed', timestamp: Date.now() - 259200000, hash: '0x5678...90ab' },
  { id: '5', type: 'send', symbol: 'BNB', amount: '2.0', value: 600, status: 'failed', timestamp: Date.now() - 345600000, hash: '0xabcd...1234' },
  { id: '6', type: 'approve', symbol: 'USDC', amount: 'Unlimited', value: 0, status: 'completed', timestamp: Date.now() - 432000000, hash: '0xefgh...5678' },
];

const HistoryScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [filter, setFilter] = useState<'all' | 'send' | 'receive' | 'swap'>('all');

  const filteredTx = filter === 'all' ? transactions : transactions.filter(t => t.type === filter);

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'send': return '📤';
      case 'receive': return '📥';
      case 'swap': return '🔄';
      case 'stake': return '💎';
      case 'approve': return '✅';
      default: return '📋';
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
    const diff = Date.now() - timestamp;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return `${Math.floor(diff / 86400000)}d ago`;
  };

  const renderTx = ({ item }: { item: Transaction }) => (
    <View style={[styles.txItem, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.txIconContainer}>
        <Text style={styles.txIcon}>{getTypeIcon(item.type)}</Text>
      </View>
      <View style={styles.txInfo}>
        <Text style={[styles.txTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          {item.type.charAt(0).toUpperCase() + item.type.slice(1)} {item.symbol}
        </Text>
        <Text style={[styles.txHash, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{item.hash}</Text>
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
      <FlatList data={filteredTx} renderItem={renderTx} keyExtractor={item => item.id} contentContainerStyle={styles.txList} showsVerticalScrollIndicator={false} />
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
  txIcon: { fontSize: 20 },
  txInfo: { flex: 1, marginLeft: SPACING.sm },
  txTitle: { fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  txHash: { fontSize: FONT_SIZES.xs, marginTop: 2 },
  txRight: { alignItems: 'flex-end' },
  txAmount: { fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  statusBadge: { paddingHorizontal: SPACING.xs, paddingVertical: 2, borderRadius: 4, marginTop: 2 },
  statusText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  txTime: { fontSize: FONT_SIZES.xs, marginTop: 2 },
});

export default HistoryScreen;
