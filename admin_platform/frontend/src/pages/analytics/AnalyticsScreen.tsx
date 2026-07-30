/**
 * TigerWallet Analytics Dashboard - Complete Implementation
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, SafeAreaView, StatusBar } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../../mobile_apps/tigerwallet/app/src/store';
import { COLORS, SPACING, FONT_SIZES } from '../../../mobile_apps/tigerwallet/app/src/constants/theme';

const AnalyticsScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [period, setPeriod] = useState<'24h' | '7d' | '30d' | '90d'>('7d');

  const stats = {
    totalVolume: 4567890000,
    totalUsers: 125430,
    totalTransactions: 1523789,
    activeWallets: 89234,
    revenue: 2345678,
    fees: 567890,
  };

  const chartData = [
    { label: 'Mon', value: 1200000 },
    { label: 'Tue', value: 1500000 },
    { label: 'Wed', value: 1100000 },
    { label: 'Thu', value: 1800000 },
    { label: 'Fri', value: 2200000 },
    { label: 'Sat', value: 1900000 },
    { label: 'Sun', value: 1600000 },
  ];

  const topChains = [
    { name: 'Ethereum', txns: 450000, volume: 2500000 },
    { name: 'BSC', txns: 380000, volume: 1200000 },
    { name: 'Polygon', txns: 280000, volume: 450000 },
    { name: 'Arbitrum', txns: 180000, volume: 280000 },
    { name: 'Solana', txns: 150000, volume: 180000 },
  ];

  const maxValue = Math.max(...chartData.map(d => d.value));

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      <View style={[styles.header, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Analytics</Text>
      </View>

      <View style={styles.periodSelector}>
        {(['24h', '7d', '30d', '90d'] as const).map(p => (
          <TouchableOpacity key={p} style={[styles.periodChip, period === p && { backgroundColor: COLORS.primary }]} onPress={() => setPeriod(p)}>
            <Text style={[styles.periodText, period === p && { color: COLORS.white }]}>{p}</Text>
          </TouchableOpacity>
        ))}
      </View>

      <ScrollView showsVerticalScrollIndicator={false}>
        <View style={styles.statsGrid}>
          <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
            <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total Volume</Text>
            <Text style={[styles.statValue, { color: COLORS.primary }]}>${(stats.totalVolume / 1000000).toFixed(2)}M</Text>
          </View>
          <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
            <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Users</Text>
            <Text style={[styles.statValue, { color: COLORS.success }]}>{(stats.totalUsers / 1000).toFixed(1)}K</Text>
          </View>
          <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
            <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Transactions</Text>
            <Text style={[styles.statValue, { color: COLORS.info }]}>{(stats.totalTransactions / 1000).toFixed(1)}K</Text>
          </View>
          <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
            <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Revenue</Text>
            <Text style={[styles.statValue, { color: COLORS.warning }]}>${(stats.revenue / 1000).toFixed(1)}K</Text>
          </View>
        </View>

        <View style={[styles.chartCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.chartTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Volume Trend</Text>
          <View style={styles.chart}>
            {chartData.map((d, i) => (
              <View key={i} style={styles.barContainer}>
                <View style={[styles.bar, { height: (d.value / maxValue) * 150, backgroundColor: COLORS.primary }]} />
                <Text style={[styles.barLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{d.label}</Text>
              </View>
            ))}
          </View>
        </View>

        <View style={[styles.chainCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.chartTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Top Chains</Text>
          {topChains.map((chain, i) => (
            <View key={i} style={styles.chainRow}>
              <Text style={[styles.chainRank, { color: COLORS.primary }]}>#{i + 1}</Text>
              <Text style={[styles.chainName, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{chain.name}</Text>
              <Text style={[styles.chainTxns, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{chain.txns.toLocaleString()} txns</Text>
              <Text style={[styles.chainVolume, { color: COLORS.success }]}>${(chain.volume / 1000).toFixed(0)}K</Text>
            </View>
          ))}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 }, header: { padding: SPACING.md }, headerTitle: { fontSize: FONT_SIZES.xl, fontWeight: 'bold' },
  periodSelector: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.md },
  periodChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.xs, borderRadius: 20, marginRight: SPACING.sm, backgroundColor: COLORS.cardDark },
  periodText: { fontSize: FONT_SIZES.sm, color: COLORS.gray }, statsGrid: { flexDirection: 'row', flexWrap: 'wrap', paddingHorizontal: SPACING.sm },
  statCard: { width: '48%', padding: SPACING.md, borderRadius: 12, margin: SPACING.xs, alignItems: 'center' },
  statLabel: { fontSize: FONT_SIZES.sm }, statValue: { fontSize: FONT_SIZES.xl, fontWeight: 'bold', marginTop: 4 },
  chartCard: { margin: SPACING.md, padding: SPACING.md, borderRadius: 12 }, chartTitle: { fontSize: FONT_SIZES.lg, fontWeight: 'bold', marginBottom: SPACING.md },
  chart: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-end', height: 180 },
  barContainer: { alignItems: 'center', flex: 1 }, bar: { width: 30, borderRadius: 4 }, barLabel: { fontSize: FONT_SIZES.xs, marginTop: 8 },
  chainCard: { margin: SPACING.md, padding: SPACING.md, borderRadius: 12 }, chainRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: SPACING.sm, borderBottomWidth: 1, borderBottomColor: COLORS.borderDark },
  chainRank: { fontSize: FONT_SIZES.sm, fontWeight: 'bold', width: 30 }, chainName: { flex: 1, fontSize: FONT_SIZES.md, fontWeight: '600' },
  chainTxns: { fontSize: FONT_SIZES.sm, marginRight: SPACING.md }, chainVolume: { fontSize: FONT_SIZES.sm, fontWeight: 'bold' },
});

export default AnalyticsScreen;
