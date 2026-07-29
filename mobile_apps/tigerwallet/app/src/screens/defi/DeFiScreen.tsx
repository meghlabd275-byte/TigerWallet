/**
 * TigerWallet DeFi Screen - Complete Implementation
 * 
 * Staking, yield farming, liquidity pools
 */

import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  SafeAreaView,
  StatusBar,
  FlatList,
} from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';

interface DeFiProduct {
  id: string;
  name: string;
  type: 'staking' | 'yield' | 'pool' | 'launchpad';
  apy: number;
  tvl: string;
  chain: string;
  logo: string;
  description: string;
  risk: 'low' | 'medium' | 'high';
}

const defiProducts: DeFiProduct[] = [
  { id: '1', name: 'ETH Staking', type: 'staking', apy: 4.5, tvl: '$12.5B', chain: 'Ethereum', logo: '🔷', description: 'Stake ETH and earn rewards', risk: 'low' },
  { id: '2', name: 'MATIC Staking', type: 'staking', apy: 8.2, tvl: '$2.1B', chain: 'Polygon', logo: '⬡', description: 'Stake MATIC for rewards', risk: 'low' },
  { id: '3', name: 'SOL Staking', type: 'staking', apy: 6.8, tvl: '$15B', chain: 'Solana', logo: '◎', description: 'Stake SOL for 6.8% APY', risk: 'low' },
  { id: '4', name: 'USDT Pool', type: 'pool', apy: 12.5, tvl: '$8.2B', chain: 'Multi', logo: '💰', description: 'Stablecoin lending pool', risk: 'medium' },
  { id: '5', name: 'ETH-USDC LP', type: 'pool', apy: 25.0, tvl: '$3.5B', chain: 'Ethereum', logo: '📊', description: 'Uniswap V3 LP position', risk: 'medium' },
  { id: '6', name: 'BNB Staking', type: 'staking', apy: 5.5, tvl: '$4.2B', chain: 'BSC', logo: '🟡', description: 'BNB Token Hub staking', risk: 'low' },
  { id: '7', name: 'AVAX Staking', type: 'staking', apy: 7.2, tvl: '$1.8B', chain: 'Avalanche', logo: '🔺', description: 'Avalanche staking', risk: 'low' },
  { id: '8', name: 'Launchpad', type: 'launchpad', apy: 0, tvl: '$500M', chain: 'Multi', logo: '🚀', description: 'Early access to new tokens', risk: 'high' },
  { id: '9', name: 'LINK Staking', type: 'staking', apy: 5.0, tvl: '$800M', chain: 'Ethereum', logo: '⛓️', description: 'Chainlink staking', risk: 'low' },
  { id: '10', name: 'Yield Optimizer', type: 'yield', apy: 18.0, tvl: '$2.5B', chain: 'Multi', logo: '📈', description: 'Auto-compound yields', risk: 'medium' },
];

const DeFiScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [selectedTab, setSelectedTab] = useState<'all' | 'staking' | 'pool' | 'yield'>('all');

  const filteredProducts = selectedTab === 'all' 
    ? defiProducts 
    : defiProducts.filter(p => p.type === selectedTab);

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'low': return COLORS.success;
      case 'medium': return COLORS.warning;
      case 'high': return COLORS.error;
      default: return COLORS.gray;
    }
  };

  const getTypeLabel = (type: string) => {
    switch (type) {
      case 'staking': return '💎 Staking';
      case 'pool': return '📊 Liquidity';
      case 'yield': return '📈 Yield';
      case 'launchpad': return '🚀 Launchpad';
      default: return type;
    }
  };

  const renderProduct = ({ item }: { item: DeFiProduct }) => (
    <TouchableOpacity style={[styles.productCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.productHeader}>
        <View style={styles.productLogoContainer}>
          <Text style={styles.productLogo}>{item.logo}</Text>
        </View>
        <View style={styles.productInfo}>
          <Text style={[styles.productName, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
            {item.name}
          </Text>
          <Text style={[styles.productChain, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
            {item.chain}
          </Text>
        </View>
        <View style={[styles.riskBadge, { backgroundColor: getRiskColor(item.risk) + '20' }]}>
          <Text style={[styles.riskText, { color: getRiskColor(item.risk) }]}>{item.risk.toUpperCase()}</Text>
        </View>
      </View>
      
      <Text style={[styles.productDescription, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
        {item.description}
      </Text>
      
      <View style={styles.productStats}>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>APY</Text>
          <Text style={[styles.statValue, { color: COLORS.success }]}>{item.apy > 0 ? `${item.apy}%` : 'N/A'}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>TVL</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.tvl}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Type</Text>
          <Text style={[styles.statValue, { color: COLORS.primary }]}>{getTypeLabel(item.type)}</Text>
        </View>
      </View>
      
      <TouchableOpacity style={[styles.stakeButton, { backgroundColor: COLORS.primary }]}>
        <Text style={styles.stakeButtonText}>{item.type === 'launchpad' ? 'View' : 'Stake'}</Text>
      </TouchableOpacity>
    </TouchableOpacity>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} backgroundColor={isDark ? COLORS.backgroundDark : COLORS.backgroundLight} />
      
      {/* Header */}
      <View style={styles.header}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          DeFi
        </Text>
        <ThemeToggle />
      </View>

      {/* Portfolio Summary */}
      <View style={[styles.summaryCard, { backgroundColor: COLORS.primary }]}>
        <Text style={styles.summaryLabel}>Total DeFi Balance</Text>
        <Text style={styles.summaryValue}>$12,450.00</Text>
        <View style={styles.summaryRow}>
          <View style={styles.summaryStat}>
            <Text style={styles.summaryStatLabel}>Total APY</Text>
            <Text style={styles.summaryStatValue}>8.5%</Text>
          </View>
          <View style={styles.summaryStat}>
            <Text style={styles.summaryStatLabel}>Active Positions</Text>
            <Text style={styles.summaryStatValue}>5</Text>
          </View>
        </View>
      </View>

      {/* Tab Filter */}
      <View style={styles.tabContainer}>
        {(['all', 'staking', 'pool', 'yield'] as const).map(tab => (
          <TouchableOpacity
            key={tab}
            style={[styles.tab, selectedTab === tab && styles.tabSelected]}
            onPress={() => setSelectedTab(tab)}
          >
            <Text style={[styles.tabText, selectedTab === tab && styles.tabTextSelected]}>
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* Products List */}
      <FlatList
        data={filteredProducts}
        renderItem={renderProduct}
        keyExtractor={item => item.id}
        contentContainerStyle={styles.productList}
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  headerTitle: { fontSize: FONT_SIZES.xxl, fontWeight: 'bold' },
  summaryCard: { margin: SPACING.md, padding: SPACING.lg, borderRadius: 16 },
  summaryLabel: { color: COLORS.white + '80', fontSize: FONT_SIZES.md },
  summaryValue: { color: COLORS.white, fontSize: 32, fontWeight: 'bold', marginVertical: SPACING.sm },
  summaryRow: { flexDirection: 'row', marginTop: SPACING.sm },
  summaryStat: { marginRight: SPACING.xl },
  summaryStatLabel: { color: COLORS.white + '80', fontSize: FONT_SIZES.sm },
  summaryStatValue: { color: COLORS.white, fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  tabContainer: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm },
  tab: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 20, marginRight: SPACING.sm },
  tabSelected: { backgroundColor: COLORS.primary },
  tabText: { fontSize: FONT_SIZES.sm, fontWeight: '600', color: COLORS.gray },
  tabTextSelected: { color: COLORS.white },
  productList: { paddingHorizontal: SPACING.md, paddingBottom: SPACING.xl },
  productCard: { padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.sm },
  productHeader: { flexDirection: 'row', alignItems: 'center' },
  productLogoContainer: { width: 44, height: 44, borderRadius: 22, backgroundColor: COLORS.primary + '20', justifyContent: 'center', alignItems: 'center' },
  productLogo: { fontSize: 22 },
  productInfo: { flex: 1, marginLeft: SPACING.sm },
  productName: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  productChain: { fontSize: FONT_SIZES.sm },
  riskBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 2, borderRadius: 4 },
  riskText: { fontSize: FONT_SIZES.xs, fontWeight: 'bold' },
  productDescription: { fontSize: FONT_SIZES.sm, marginTop: SPACING.sm },
  productStats: { flexDirection: 'row', justifyContent: 'space-between', marginTop: SPACING.md },
  statItem: {},
  statLabel: { fontSize: FONT_SIZES.xs },
  statValue: { fontSize: FONT_SIZES.md, fontWeight: 'bold', marginTop: 2 },
  stakeButton: { marginTop: SPACING.md, padding: SPACING.sm, borderRadius: 8, alignItems: 'center' },
  stakeButtonText: { color: COLORS.white, fontSize: FONT_SIZES.md, fontWeight: 'bold' },
});

export default DeFiScreen;
