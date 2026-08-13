/**
 * TigerWallet Wallet Screen - Complete Implementation
 * 
 * Main wallet view with multi-chain balances, tokens, and actions
 */

import React, { useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  RefreshControl,
  SafeAreaView,
  StatusBar,
  FlatList,
  Image,
} from 'react-native';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../../store';
import { setSelectedChain, setBalances } from '../../store/slices/walletSlice';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ChainId, NATIVE_CURRENCIES } from '../../constants/chains';
import { ThemeToggle } from '../../components/ThemeToggle';
import { API } from '../../services/API';

interface TokenBalance {
  symbol: string;
  name: string;
  balance: string;
  value: number;
  logo?: string;
  chainId: number;
}

const WalletScreen: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const theme = useSelector((state: RootState) => state.theme.mode);
  const wallet = useSelector((state: RootState) => state.wallet.wallet);
  const selectedChainId = useSelector((state: RootState) => state.wallet.selectedChainId);
  const isDark = theme === 'dark';

  const [refreshing, setRefreshing] = useState(false);
  const [totalValue, setTotalValue] = useState(0);
  const [tokens, setTokens] = useState<TokenBalance[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch real token balances + native balance for the active wallet on the
  // selected chain from the canonical wallet_api backend (no hardcoded data).
  const loadBalances = async () => {
    if (!wallet?.id) {
      setTokens([]);
      setTotalValue(0);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const nativeRes = await API.getBalance(wallet.id, selectedChainId);
      const tokensRes = await API.getTokens(wallet.id, selectedChainId);

      const nativeBal = nativeRes?.data?.balance ?? '0';
      const nativeSymbol = NATIVE_CURRENCIES[selectedChainId]?.symbol || 'ETH';
      const built: TokenBalance[] = [];

      // Native coin row (only show if there is a balance)
      if (parseFloat(nativeBal) > 0) {
        const priceRes = await API.getPrice(nativeSymbol.toLowerCase());
        const price = priceRes?.data?.price ?? 0;
        built.push({
          symbol: nativeSymbol,
          name: NATIVE_CURRENCIES[selectedChainId]?.name || nativeSymbol,
          balance: nativeBal,
          value: parseFloat(nativeBal) * price,
          chainId: selectedChainId,
        });
      }

      // ERC-20 rows returned by the backend
      const tokenList = tokensRes?.data?.tokens ?? tokensRes?.data ?? [];
      for (const t of tokenList) {
        const sym = t.symbol || t.contractAddress?.slice(0, 4) || 'TOKEN';
        const bal = t.balance ?? t.amount ?? '0';
        const priceRes = await API.getPrice(sym.toLowerCase());
        const price = priceRes?.data?.price ?? 0;
        built.push({
          symbol: sym,
          name: t.name || sym,
          balance: bal,
          value: parseFloat(bal) * price,
          chainId: selectedChainId,
        });
      }

      setTokens(built);
      setTotalValue(built.reduce((sum, token) => sum + token.value, 0));
      dispatch(setBalances({ [selectedChainId]: nativeBal }));
    } catch (err) {
      setError('Failed to load balances. Pull to retry.');
      setTokens([]);
      setTotalValue(0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadBalances();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wallet?.id, selectedChainId]);

  const onRefresh = async () => {
    setRefreshing(true);
    await loadBalances();
    setRefreshing(false);
  };

  const handleChainSelect = (chainId: number) => {
    dispatch(setSelectedChain(chainId));
  };

  const renderTokenItem = ({ item }: { item: TokenBalance }) => (
    <TouchableOpacity style={[styles.tokenItem, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={styles.tokenLeft}>
        <View style={[styles.tokenIcon, { backgroundColor: COLORS.primary + '20' }]}>
          <Text style={styles.tokenIconText}>{item.symbol.charAt(0)}</Text>
        </View>
        <View>
          <Text style={[styles.tokenSymbol, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
            {item.symbol}
          </Text>
          <Text style={[styles.tokenName, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
            {item.name}
          </Text>
        </View>
      </View>
      <View style={styles.tokenRight}>
        <Text style={[styles.tokenBalance, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          {item.balance}
        </Text>
        <Text style={[styles.tokenValue, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
          ${item.value.toLocaleString()}
        </Text>
      </View>
    </TouchableOpacity>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} backgroundColor={isDark ? COLORS.backgroundDark : COLORS.backgroundLight} />
      
      {/* Header */}
      <View style={styles.header}>
        <View>
          <Text style={[styles.greeting, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
            Total Balance
          </Text>
          <Text style={[styles.totalBalance, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
            ${totalValue.toLocaleString()}
          </Text>
        </View>
        <ThemeToggle />
      </View>

      {/* Quick Actions */}
      <View style={styles.actions}>
        <TouchableOpacity style={[styles.actionButton, { backgroundColor: COLORS.primary }]}>
          <Text style={styles.actionIcon}>📥</Text>
          <Text style={styles.actionText}>Receive</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionButton, { backgroundColor: COLORS.primary }]}>
          <Text style={styles.actionIcon}>📤</Text>
          <Text style={styles.actionText}>Send</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionButton, { backgroundColor: COLORS.primary }]}>
          <Text style={styles.actionIcon}>🔄</Text>
          <Text style={styles.actionText}>Swap</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionButton, { backgroundColor: COLORS.primary }]}>
          <Text style={styles.actionIcon}>📊</Text>
          <Text style={styles.actionText}>Buy</Text>
        </TouchableOpacity>
      </View>

      {/* Chain Selector */}
      <View style={styles.chainSelector}>
        <ScrollView horizontal showsHorizontalScrollIndicator={false}>
          {[1, 56, 137, 42161, 10, 43114, 8453, 501].map(chainId => (
            <TouchableOpacity
              key={chainId}
              style={[
                styles.chainChip,
                { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight },
                selectedChainId === chainId && styles.chainChipSelected
              ]}
              onPress={() => handleChainSelect(chainId)}
            >
              <Text style={[
                styles.chainChipText,
                { color: isDark ? COLORS.textDark : COLORS.textLight },
                selectedChainId === chainId && styles.chainChipTextSelected
              ]}>
                {NATIVE_CURRENCIES[chainId]?.symbol || 'UNK'}
              </Text>
            </TouchableOpacity>
          ))}
        </ScrollView>
      </View>

      {/* Token List */}
      <View style={styles.tokenListHeader}>
        <Text style={[styles.tokenListTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          Assets
        </Text>
        <TouchableOpacity>
          <Text style={[styles.addTokenText, { color: COLORS.primary }]}>+ Add Token</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={tokens}
        renderItem={renderTokenItem}
        keyExtractor={item => item.symbol}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={COLORS.primary} />}
        contentContainerStyle={styles.tokenList}
        showsVerticalScrollIndicator={false}
        ListEmptyComponent={
          <View style={styles.emptyContainer}>
            <Text style={{ color: isDark ? COLORS.gray : COLORS.lightGray, textAlign: 'center' }}>
              {loading ? 'Loading balances...' : error || 'No tokens found. Pull to refresh.'}
            </Text>
          </View>
        }
      />
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  greeting: { fontSize: FONT_SIZES.md },
  totalBalance: { fontSize: 36, fontWeight: 'bold' },
  actions: { flexDirection: 'row', justifyContent: 'space-around', paddingVertical: SPACING.md, paddingHorizontal: SPACING.sm },
  actionButton: { alignItems: 'center', padding: SPACING.sm, borderRadius: 12, minWidth: 70 },
  actionIcon: { fontSize: 24, marginBottom: 4 },
  actionText: { color: COLORS.white, fontSize: FONT_SIZES.sm, fontWeight: '600' },
  chainSelector: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm },
  chainChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 20, marginRight: SPACING.sm },
  chainChipSelected: { backgroundColor: COLORS.primary },
  chainChipText: { fontSize: FONT_SIZES.sm, fontWeight: '600' },
  chainChipTextSelected: { color: COLORS.white },
  tokenListHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm },
  tokenListTitle: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  addTokenText: { fontSize: FONT_SIZES.md, fontWeight: '600' },
  tokenList: { paddingHorizontal: SPACING.md, paddingBottom: SPACING.xl },
  tokenItem: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.sm },
  tokenLeft: { flexDirection: 'row', alignItems: 'center' },
  tokenIcon: { width: 40, height: 40, borderRadius: 20, justifyContent: 'center', alignItems: 'center', marginRight: SPACING.sm },
  tokenIconText: { fontSize: FONT_SIZES.lg, fontWeight: 'bold', color: COLORS.primary },
  tokenSymbol: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  tokenName: { fontSize: FONT_SIZES.sm },
  tokenRight: { alignItems: 'flex-end' },
  tokenBalance: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  tokenValue: { fontSize: FONT_SIZES.sm },
  emptyContainer: { padding: SPACING.xl, alignItems: 'center' },
});

export default WalletScreen;
