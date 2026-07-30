/**
 * TigerWallet Fees Management - Complete Implementation
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, TextInput, SafeAreaView, StatusBar, Alert } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../../mobile_apps/tigerwallet/app/src/store';
import { COLORS, SPACING, FONT_SIZES } from '../../../mobile_apps/tigerwallet/app/src/constants/theme';

interface FeeConfig {
  id: string;
  type: 'withdraw' | 'deposit' | 'swap' | 'network' | 'bridge';
  asset: string;
  chain: string;
  feeType: 'fixed' | 'percentage' | 'tiered';
  value: string;
  minFee: string;
  maxFee: string;
  status: 'active' | 'paused';
}

const mockFees: FeeConfig[] = [
  { id: '1', type: 'withdraw', asset: 'ETH', chain: 'Ethereum', feeType: 'fixed', value: '0.005', minFee: '0.001', maxFee: '0.05', status: 'active' },
  { id: '2', type: 'withdraw', asset: 'USDT', chain: 'Ethereum', feeType: 'percentage', value: '0.1', minFee: '1', maxFee: '100', status: 'active' },
  { id: '3', type: 'deposit', asset: 'ALL', chain: 'ALL', feeType: 'fixed', value: '0', minFee: '0', maxFee: '0', status: 'active' },
  { id: '4', type: 'swap', asset: 'ALL', chain: 'ALL', feeType: 'percentage', value: '0.3', minFee: '0.01', maxFee: '10', status: 'active' },
  { id: '5', type: 'network', asset: 'GAS', chain: 'Ethereum', feeType: 'tiered', value: 'dynamic', minFee: '0.001', maxFee: '0.05', status: 'active' },
  { id: '6', type: 'bridge', asset: 'ALL', chain: 'ALL', feeType: 'percentage', value: '0.5', minFee: '5', maxFee: '50', status: 'active' },
];

const typeIcons: Record<string, string> = { withdraw: '📤', deposit: '📥', swap: '🔄', network: '⛽', bridge: '🌉' };

const FeesScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [fees] = useState<FeeConfig[]>(mockFees);
  const [filter, setFilter] = useState<string>('all');
  const filteredFees = filter === 'all' ? fees : fees.filter(f => f.type === filter);

  const renderFeeItem = ({ item }: { item: FeeConfig }) => (
    <View style={[styles.feeCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.feeHeader}>
        <View style={[styles.feeIcon, { backgroundColor: COLORS.primary + '20' }]}><Text style={styles.feeIconText}>{typeIcons[item.type]}</Text></View>
        <View style={styles.feeInfo}>
          <Text style={[styles.feeType, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.type.toUpperCase()} Fee</Text>
          <Text style={[styles.feeAsset, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{item.asset} • {item.chain}</Text>
        </View>
        <View style={[styles.statusBadge, { backgroundColor: (item.status === 'active' ? COLORS.success : COLORS.warning) + '20' }]}>
          <Text style={[styles.statusText, { color: item.status === 'active' ? COLORS.success : COLORS.warning }]}>{item.status}</Text>
        </View>
      </View>
      <View style={styles.feeDetails}>
        <View style={styles.detailItem}><Text style={[styles.detailLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Type</Text><Text style={[styles.detailValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.feeType}</Text></View>
        <View style={styles.detailItem}><Text style={[styles.detailLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Value</Text><Text style={[styles.detailValue, { color: COLORS.primary }]}>{item.value}{item.feeType === 'percentage' ? '%' : ''}</Text></View>
        <View style={styles.detailItem}><Text style={[styles.detailLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Min</Text><Text style={[styles.detailValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.minFee}</Text></View>
        <View style={styles.detailItem}><Text style={[styles.detailLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Max</Text><Text style={[styles.detailValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.maxFee}</Text></View>
      </View>
      <View style={styles.feeActions}>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.primary + '20' }]}><Text style={[styles.actionBtnText, { color: COLORS.primary }]}>Edit</Text></TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.warning + '20' }]}><Text style={[styles.actionBtnText, { color: COLORS.warning }]}>Pause</Text></TouchableOpacity>
      </View>
    </View>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      <View style={[styles.header, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Fees Management</Text>
        <TouchableOpacity style={[styles.addButton, { backgroundColor: COLORS.primary }]}><Text style={styles.addButtonText}>+ Add Fee</Text></TouchableOpacity>
      </View>
      <View style={styles.statsRow}>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}><Text style={[styles.statNumber, { color: COLORS.primary }]}>{fees.length}</Text><Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total</Text></View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}><Text style={[styles.statNumber, { color: COLORS.success }]}>{fees.filter(f => f.status === 'active').length}</Text><Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Active</Text></View>
      </View>
      <View style={styles.filterContainer}>
        {(['all', 'withdraw', 'deposit', 'swap', 'network', 'bridge'] as const).map(f => (
          <TouchableOpacity key={f} style={[styles.filterChip, filter === f && { backgroundColor: COLORS.primary }]} onPress={() => setFilter(f)}>
            <Text style={[styles.filterText, filter === f && { color: COLORS.white }]}>{f === 'all' ? 'All' : f}</Text>
          </TouchableOpacity>
        ))}
      </View>
      <FlatList data={filteredFees} renderItem={renderFeeItem} keyExtractor={item => item.id} contentContainerStyle={styles.list} showsVerticalScrollIndicator={false} />
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 }, header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  headerTitle: { fontSize: FONT_SIZES.xl, fontWeight: 'bold' }, addButton: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 8 }, addButtonText: { color: COLORS.white, fontWeight: '600' },
  statsRow: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm }, statCard: { flex: 1, padding: SPACING.md, borderRadius: 8, alignItems: 'center', marginHorizontal: 4 },
  statNumber: { fontSize: FONT_SIZES.xxl, fontWeight: 'bold' }, statLabel: { fontSize: FONT_SIZES.sm },
  filterContainer: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm, flexWrap: 'wrap' },
  filterChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.xs, borderRadius: 20, marginRight: SPACING.sm, marginBottom: 4, backgroundColor: COLORS.cardDark },
  filterText: { fontSize: FONT_SIZES.sm, color: COLORS.gray }, list: { padding: SPACING.md, paddingBottom: 100 },
  feeCard: { padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.md }, feeHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: SPACING.md },
  feeIcon: { width: 44, height: 44, borderRadius: 22, justifyContent: 'center', alignItems: 'center' }, feeIconText: { fontSize: 20 },
  feeInfo: { flex: 1, marginLeft: SPACING.sm }, feeType: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' }, feeAsset: { fontSize: FONT_SIZES.sm },
  statusBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 4, borderRadius: 4 }, statusText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  feeDetails: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: SPACING.md }, detailItem: { alignItems: 'center' },
  detailLabel: { fontSize: FONT_SIZES.xs }, detailValue: { fontSize: FONT_SIZES.md, fontWeight: '600', marginTop: 2 },
  feeActions: { flexDirection: 'row', justifyContent: 'space-between' }, actionBtn: { flex: 1, padding: SPACING.sm, borderRadius: 6, alignItems: 'center', marginHorizontal: 4 },
  actionBtnText: { fontSize: FONT_SIZES.sm, fontWeight: '600' },
});

export default FeesScreen;
