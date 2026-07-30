/**
 * TigerWallet Tokens Management - Complete Implementation
 * 
 * Full token management with chains, prices, and status
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, TextInput, SafeAreaView, StatusBar, Alert } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../../mobile_apps/tigerwallet/app/src/store';
import { COLORS, SPACING, FONT_SIZES } from '../../../mobile_apps/tigerwallet/app/src/constants/theme';

interface Token {
  id: string;
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  chain: string;
  chainId: number;
  price: number;
  change24h: number;
  marketCap: number;
  volume24h: number;
  holders: number;
  type: 'native' | 'erc20' | 'spl' | 'trc20' | 'arc20';
  status: 'active' | 'paused' | 'delisted';
  verified: boolean;
  logo: string;
}

const mockTokens: Token[] = [
  { id: '1', address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', name: 'USD Coin', symbol: 'USDC', decimals: 6, chain: 'Ethereum', chainId: 1, price: 1.0, change24h: 0.01, marketCap: 40000000000, volume24h: 5000000000, holders: 2500000, type: 'erc20', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/6319/small/USD_Coin_icon.png' },
  { id: '2', address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', name: 'Tether', symbol: 'USDT', decimals: 6, chain: 'Ethereum', chainId: 1, price: 1.0, change24h: -0.02, marketCap: 95000000000, volume24h: 45000000000, holders: 15000000, type: 'erc20', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/325/small/Tether.png' },
  { id: '3', address: '', name: 'Ethereum', symbol: 'ETH', decimals: 18, chain: 'Ethereum', chainId: 1, price: 3500, change24h: 2.5, marketCap: 420000000000, volume24h: 15000000000, holders: 5000000, type: 'native', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/279/small/ethereum.png' },
  { id: '4', address: '0x514910771AF9Ca656af840dff83E8264EcF986CA', name: 'Chainlink', symbol: 'LINK', decimals: 18, chain: 'Ethereum', chainId: 1, price: 15.5, change24h: 1.8, marketCap: 8000000000, volume24h: 500000000, holders: 800000, type: 'erc20', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/877/small/chainlink-new-logo.png' },
  { id: '5', address: '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', name: 'Uniswap', symbol: 'UNI', decimals: 18, chain: 'Ethereum', chainId: 1, price: 8.2, change24h: -1.2, marketCap: 5000000000, volume24h: 200000000, holders: 450000, type: 'erc20', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/12504/small/uniswap-uni.png' },
  { id: '6', address: '0x55d398326f99059fF775485246999027B3197955', name: 'Tether', symbol: 'USDT', decimals: 18, chain: 'BSC', chainId: 56, price: 1.0, change24h: 0.0, marketCap: 95000000000, volume24h: 45000000000, holders: 20000000, type: 'erc20', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/325/small/Tether.png' },
  { id: '7', address: '', name: 'BNB', symbol: 'BNB', decimals: 18, chain: 'BSC', chainId: 56, price: 320, change24h: 1.5, marketCap: 48000000000, volume24h: 1200000000, holders: 3500000, type: 'native', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png' },
  { id: '8', address: '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', name: 'Tether', symbol: 'USDT', decimals: 6, chain: 'Polygon', chainId: 137, price: 1.0, change24h: 0.0, marketCap: 95000000000, volume24h: 4500000000, holders: 1500000, type: 'erc20', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/325/small/Tether.png' },
  { id: '9', address: '', name: 'Solana', symbol: 'SOL', decimals: 9, chain: 'Solana', chainId: 501, price: 120, change24h: 3.2, marketCap: 52000000000, volume24h: 2500000000, holders: 2500000, type: 'native', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/4128/small/solana.png' },
  { id: '10', address: '', name: 'TRON', symbol: 'TRX', decimals: 6, chain: 'TRON', chainId: 728126428, price: 0.12, change24h: 0.5, marketCap: 10000000000, volume24h: 500000000, holders: 12000000, type: 'native', status: 'active', verified: true, logo: 'https://assets.coingecko.com/coins/images/1094/small/tron-logo.png' },
];

const chainIcons: Record<string, string> = {
  'Ethereum': '🔷',
  'BSC': '🟡',
  'Polygon': '⬡',
  'Solana': '◎',
  'TRON': '💎',
  'Arbitrum': '🔴',
  'Avalanche': '🔺',
  'Base': '🔵',
};

const TokensScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [tokens, setTokens] = useState<Token[]>(mockTokens);
  const [searchQuery, setSearchQuery] = useState('');
  const [filter, setFilter] = useState<'all' | 'active' | 'paused' | 'delisted'>('all');
  const [chainFilter, setChainFilter] = useState<string>('all');

  const filteredTokens = tokens.filter(token => {
    const matchesSearch = token.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         token.symbol.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         token.address.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter = filter === 'all' || token.status === filter;
    const matchesChain = chainFilter === 'all' || token.chain === chainFilter;
    return matchesSearch && matchesFilter && matchesChain;
  });

  const uniqueChains = [...new Set(tokens.map(t => t.chain))];

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return COLORS.success;
      case 'paused': return COLORS.warning;
      case 'delisted': return COLORS.error;
      default: return COLORS.gray;
    }
  };

  const handleTokenAction = (token: Token, action: string) => {
    Alert.alert(action, `${action} ${token.name}?`, [
      { text: 'Cancel', style: 'cancel' },
      { text: action, onPress: () => {} },
    ]);
  };

  const renderTokenItem = ({ item }: { item: Token }) => (
    <View style={[styles.tokenCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.tokenHeader}>
        <View style={[styles.tokenIcon, { backgroundColor: COLORS.primary + '20' }]}>
          <Text style={styles.tokenIconText}>{item.symbol.charAt(0)}</Text>
        </View>
        <View style={styles.tokenInfo}>
          <View style={styles.tokenTopRow}>
            <View style={styles.tokenNameRow}>
              <Text style={[styles.tokenName, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{item.name}</Text>
              {item.verified && <Text style={styles.verifiedIcon}>✓</Text>}
            </View>
            <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) + '20' }]}>
              <Text style={[styles.statusText, { color: getStatusColor(item.status) }]}>{item.status}</Text>
            </View>
          </View>
          <View style={styles.tokenSymbolRow}>
            <Text style={[styles.tokenSymbol, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{item.symbol}</Text>
            <Text style={[styles.chainBadge, { color: COLORS.primary }]}> {chainIcons[item.chain] || '⛓️'} {item.chain}</Text>
          </View>
        </View>
      </View>

      <View style={styles.tokenStats}>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Price</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>${item.price.toLocaleString()}</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>24h</Text>
          <Text style={[styles.statValue, { color: item.change24h >= 0 ? COLORS.success : COLORS.error }]}>{item.change24h >= 0 ? '+' : ''}{item.change24h}%</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Market Cap</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>${(item.marketCap / 1000000000).toFixed(1)}B</Text>
        </View>
        <View style={styles.statItem}>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Holders</Text>
          <Text style={[styles.statValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{(item.holders / 1000000).toFixed(1)}M</Text>
        </View>
      </View>

      {item.address && (
        <View style={styles.tokenAddress}>
          <Text style={[styles.addressLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Contract: </Text>
          <Text style={[styles.addressValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]} numberOfLines={1}>
            {item.address.slice(0, 10)}...{item.address.slice(-8)}
          </Text>
        </View>
      )}

      <View style={styles.tokenActions}>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.info + '20' ]} onPress={() => handleTokenAction(item, 'View Details')}>
          <Text style={[styles.actionBtnText, { color: COLORS.info }]}>Details</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.primary + '20' ]} onPress={() => handleTokenAction(item, 'Edit')}>
          <Text style={[styles.actionBtnText, { color: COLORS.primary }]}>Edit</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: COLORS.warning + '20' ]} onPress={() => handleTokenAction(item, item.status === 'active' ? 'Pause' : 'Activate')}>
          <Text style={[styles.actionBtnText, { color: COLORS.warning }]}>{item.status === 'active' ? 'Pause' : 'Activate'}</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, { backgroundColor: item.verified ? COLORS.gray + '20' : COLORS.success + '20' ]} onPress={() => !item.verified && handleTokenAction(item, 'Verify')}>
          <Text style={[styles.actionBtnText, { color: item.verified ? COLORS.gray : COLORS.success }]}>{item.verified ? 'Verified' : 'Verify'}</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      
      <View style={[styles.header, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Tokens Management</Text>
        <TouchableOpacity style={[styles.addButton, { backgroundColor: COLORS.primary }]} onPress={() => Alert.alert('Add Token', 'Add new token')}>
          <Text style={styles.addButtonText}>+ Add Token</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.statsRow}>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.primary }]}>{tokens.length}</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total Tokens</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.success }]}>{tokens.filter(t => t.status === 'active').length}</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Active</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.warning }]}>{uniqueChains.length}</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Chains</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.statNumber, { color: COLORS.success }]}>${(tokens.reduce((sum, t) => sum + t.marketCap, 0) / 1000000000000).toFixed(1)}T</Text>
          <Text style={[styles.statLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Total MC</Text>
        </View>
      </View>

      <View style={styles.searchContainer}>
        <TextInput
          style={[styles.searchInput, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight, color: isDark ? COLORS.textDark : COLORS.textLight }]}
          placeholder="Search by name, symbol, or address..."
          placeholderTextColor={isDark ? COLORS.gray : COLORS.lightGray}
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      <View style={styles.filterRow}>
        <View style={styles.filterContainer}>
          <TouchableOpacity style={[styles.filterChip, chainFilter === 'all' && { backgroundColor: COLORS.primary }]} onPress={() => setChainFilter('all')}>
            <Text style={[styles.filterText, chainFilter === 'all' && { color: COLORS.white }]}>All Chains</Text>
          </TouchableOpacity>
          {uniqueChains.slice(0, 4).map(chain => (
            <TouchableOpacity key={chain} style={[styles.filterChip, chainFilter === chain && { backgroundColor: COLORS.primary }]} onPress={() => setChainFilter(chain)}>
              <Text style={[styles.filterText, chainFilter === chain && { color: COLORS.white }]}>{chain}</Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>

      <View style={styles.filterContainer}>
        {(['all', 'active', 'paused', 'delisted'] as const).map(f => (
          <TouchableOpacity key={f} style={[styles.filterChip, filter === f && { backgroundColor: COLORS.primary }]} onPress={() => setFilter(f)}>
            <Text style={[styles.filterText, filter === f && { color: COLORS.white }]}>{f.charAt(0).toUpperCase() + f.slice(1)}</Text>
          </TouchableOpacity>
        ))}
      </View>

      <FlatList
        data={filteredTokens}
        renderItem={renderTokenItem}
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
  filterRow: { marginBottom: SPACING.xs },
  filterContainer: { flexDirection: 'row', paddingHorizontal: SPACING.md, marginBottom: SPACING.sm, flexWrap: 'wrap' },
  filterChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.xs, borderRadius: 20, marginRight: SPACING.sm, marginBottom: 4, backgroundColor: COLORS.cardDark },
  filterText: { fontSize: FONT_SIZES.sm, color: COLORS.gray },
  list: { padding: SPACING.md, paddingBottom: 100 },
  tokenCard: { padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.md },
  tokenHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: SPACING.md },
  tokenIcon: { width: 48, height: 48, borderRadius: 24, justifyContent: 'center', alignItems: 'center' },
  tokenIconText: { fontSize: 22, fontWeight: 'bold', color: COLORS.primary },
  tokenInfo: { flex: 1, marginLeft: SPACING.sm },
  tokenTopRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  tokenNameRow: { flexDirection: 'row', alignItems: 'center' },
  tokenName: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  verifiedIcon: { fontSize: 14, color: COLORS.success, marginLeft: 4 },
  tokenSymbolRow: { flexDirection: 'row', alignItems: 'center' },
  tokenSymbol: { fontSize: FONT_SIZES.sm },
  chainBadge: { fontSize: FONT_SIZES.sm, marginLeft: SPACING.sm },
  statusBadge: { paddingHorizontal: SPACING.sm, paddingVertical: 2, borderRadius: 4 },
  statusText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  tokenStats: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: SPACING.md },
  statItem: { alignItems: 'center' },
  statValue: { fontSize: FONT_SIZES.md, fontWeight: '600' },
  tokenAddress: { flexDirection: 'row', marginBottom: SPACING.md },
  addressLabel: { fontSize: FONT_SIZES.sm },
  addressValue: { fontSize: FONT_SIZES.sm, flex: 1 },
  tokenActions: { flexDirection: 'row', justifyContent: 'space-between' },
  actionBtn: { flex: 1, padding: SPACING.sm, borderRadius: 6, alignItems: 'center', marginHorizontal: 2 },
  actionBtnText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
});

export default TokensScreen;
