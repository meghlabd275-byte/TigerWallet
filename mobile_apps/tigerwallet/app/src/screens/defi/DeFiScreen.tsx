/**
 * TigerWallet DeFi Screen - Complete Implementation
 * 
 * Staking, yield farming, liquidity pools
 */

import React, { useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  SafeAreaView,
  StatusBar,
  FlatList,
  ActivityIndicator,
} from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';
import { API } from '../../services/API';

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

const DeFiScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [selectedTab, setSelectedTab] = useState<'all' | 'staking' | 'pool' | 'yield'>('all');
  const [products, setProducts] = useState<DeFiProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [portfolio, setPortfolio] = useState<{ totalValue: number; totalApy: number; activePositions: number }>({
    totalValue: 0,
    totalApy: 0,
    activePositions: 0,
  });

  // Fetch real DeFi earn products + portfolio from the backend (no hardcoded
  // product list). earn_service :8466 + wallet_api portfolio endpoint.
  const loadDeFi = async () => {
    setLoading(true);
    setError(null);
    try {
      const [productsRes, portfolioRes] = await Promise.all([
        API.getEarnProducts(),
        API.getPortfolio().catch(() => null),
      ]);

      const list = productsRes?.data?.products ?? productsRes?.data ?? [];
      const mapped: DeFiProduct[] = (list as any[]).map((p) => ({
        id: p.id ?? p.product_id ?? p.name,
        name: p.name ?? p.asset ?? 'Product',
        type: (p.type ?? p.category ?? 'staking') as DeFiProduct['type'],
        apy: (p.apy ?? p.apr ?? 0) as number,
        tvl: p.tvl ?? p.total_staked ?? '—',
        chain: p.chain ?? p.network ?? 'Multi',
        logo: '💎',
        description: p.description ?? '',
        risk: (p.risk ?? 'medium') as DeFiProduct['risk'],
      }));
      setProducts(mapped);

      if (portfolioRes?.data) {
        setPortfolio({
          totalValue: portfolioRes.data.totalValue ?? portfolioRes.data.total_value ?? 0,
          totalApy: portfolioRes.data.totalApy ?? portfolioRes.data.total_apy ?? 0,
          activePositions: portfolioRes.data.activePositions ?? portfolioRes.data.active_positions ?? mapped.length,
        });
      } else {
        setPortfolio((prev) => ({ ...prev, activePositions: mapped.length }));
      }
    } catch (err) {
      setError('Failed to load DeFi products. Pull down to retry.');
      setProducts([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDeFi();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filteredProducts = selectedTab === 'all'
    ? products
    : products.filter(p => p.type === selectedTab);

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
        <Text style={styles.summaryValue}>${portfolio.totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</Text>
        <View style={styles.summaryRow}>
          <View style={styles.summaryStat}>
            <Text style={styles.summaryStatLabel}>Total APY</Text>
            <Text style={styles.summaryStatValue}>{portfolio.totalApy.toFixed(1)}%</Text>
          </View>
          <View style={styles.summaryStat}>
            <Text style={styles.summaryStatLabel}>Active Positions</Text>
            <Text style={styles.summaryStatValue}>{portfolio.activePositions}</Text>
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
        refreshControl={undefined}
        ListEmptyComponent={
          <View style={styles.emptyContainer}>
            {loading ? (
              <ActivityIndicator color={COLORS.primary} />
            ) : (
              <Text style={{ color: isDark ? COLORS.gray : COLORS.lightGray, textAlign: 'center' }}>
                {error || 'No DeFi products available.'}
              </Text>
            )}
            {!loading && error && (
              <TouchableOpacity style={styles.retryButton} onPress={loadDeFi}>
                <Text style={styles.retryButtonText}>Retry</Text>
              </TouchableOpacity>
            )}
          </View>
        }
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
  emptyContainer: { padding: SPACING.xl, alignItems: 'center' },
  retryButton: { marginTop: SPACING.md, backgroundColor: COLORS.primary, paddingHorizontal: 24, paddingVertical: 10, borderRadius: 8 },
  retryButtonText: { color: COLORS.white, fontWeight: '600' },
});

export default DeFiScreen;
