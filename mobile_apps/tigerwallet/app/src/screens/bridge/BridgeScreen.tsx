/**
 * TigerWallet Bridge Screen - Cross-chain bridge interface
 *
 * Fetches real bridge quotes from the canonical wallet_api bridge endpoints
 * (no hardcoded bridge list). The bridge aggregator routes to the best
 * available cross-chain protocol.
 */

import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, TextInput, SafeAreaView, StatusBar, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';
import { API } from '../../services/API';

interface ChainOption {
  id: number;
  name: string;
  symbol: string;
}

interface BridgeQuote {
  bridge?: string;
  bridgeName?: string;
  outputAmount?: string;
  toAmount?: string;
  estimatedTime?: string;
  estimated_time?: string;
  fee?: string;
  route?: string;
}

const BridgeScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const wallet = useSelector((state: RootState) => state.wallet.wallet);
  const isDark = theme === 'dark';

  const [chains, setChains] = useState<ChainOption[]>([]);
  const [fromChainId, setFromChainId] = useState(1);
  const [toChainId, setToChainId] = useState(137);
  const [amount, setAmount] = useState('');
  const [quote, setQuote] = useState<BridgeQuote | null>(null);
  const [loading, setLoading] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load the live chain registry from the backend (120 EVM mainnet chains).
  useEffect(() => {
    const loadChains = async () => {
      try {
        const res = await API.getChains('evm');
        const list = res?.data?.chains ?? res?.data ?? [];
        const mapped: ChainOption[] = (list as any[])
          .filter((c) => !c.is_testnet)
          .map((c) => ({
            id: c.chain_id ?? c.id,
            name: c.name,
            symbol: c.native_currency?.symbol ?? c.symbol ?? 'ETH',
          }));
        if (mapped.length > 0) setChains(mapped);
      } catch (err) {
        // Fail-closed: keep the default chain selection if backend is unreachable.
      }
    };
    loadChains();
  }, []);

  // Fetch a real bridge quote whenever the inputs change (debounced).
  useEffect(() => {
    if (!amount || parseFloat(amount) <= 0 || fromChainId === toChainId) {
      setQuote(null);
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    const timer = setTimeout(async () => {
      try {
        const res = await API.getBridgeQuotes({
          fromChainId,
          toChainId,
          fromToken: 'ETH',
          toToken: 'ETH',
          amount,
          fromAddress: wallet?.addresses?.[fromChainId] || '',
        });
        if (!cancelled) {
          if (res?.success === false) {
            setError(res.error || 'No bridge route found');
            setQuote(null);
          } else {
            setQuote(res?.data);
          }
        }
      } catch (err) {
        if (!cancelled) setError('Failed to fetch bridge quote');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }, 400);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [amount, fromChainId, toChainId, wallet]);

  const handleBridge = async () => {
    if (!wallet?.id || !quote) {
      Alert.alert('Error', 'Enter an amount and fetch a quote first');
      return;
    }
    setExecuting(true);
    try {
      const res = await API.executeBridge({
        quoteId: (quote as any)?.quoteId ?? (quote as any)?.quote_id ?? (quote as any)?.id ?? '',
        walletId: wallet.id,
        toAddress: wallet?.addresses?.[toChainId] || '',
      });
      if (res?.success === false) {
        Alert.alert('Error', res.error || 'Bridge failed');
      } else {
        Alert.alert('Success', 'Bridge transaction submitted!');
        setAmount('');
        setQuote(null);
      }
    } catch (err) {
      Alert.alert('Error', 'Bridge request failed');
    } finally {
      setExecuting(false);
    }
  };

  const chainOptions = chains.length > 0 ? chains : [
    { id: 1, name: 'Ethereum', symbol: 'ETH' },
    { id: 56, name: 'BSC', symbol: 'BNB' },
    { id: 137, name: 'Polygon', symbol: 'MATIC' },
    { id: 42161, name: 'Arbitrum', symbol: 'ETH' },
    { id: 10, name: 'Optimism', symbol: 'ETH' },
    { id: 43114, name: 'Avalanche', symbol: 'AVAX' },
    { id: 8453, name: 'Base', symbol: 'ETH' },
  ];

  const estimatedTime = quote?.estimatedTime ?? quote?.estimated_time ?? (fromChainId === toChainId ? 'Same chain' : '~10-30 minutes');

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      <View style={styles.header}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Bridge</Text>
        <ThemeToggle />
      </View>

      <ScrollView contentContainerStyle={styles.content}>
        {/* From Chain */}
        <View style={[styles.chainCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.chainLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>From</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
            {chainOptions.map(chain => (
              <TouchableOpacity key={`from-${chain.id}`} style={[styles.chainChip, fromChainId === chain.id && styles.chainChipSelected]} onPress={() => setFromChainId(chain.id)}>
                <Text style={[styles.chainChipText, fromChainId === chain.id && styles.chainChipTextSelected]}>{chain.name}</Text>
              </TouchableOpacity>
            ))}
          </ScrollView>
          <View style={styles.amountContainer}>
            <TextInput
              style={[styles.amountInput, { color: isDark ? COLORS.textDark : COLORS.textLight, backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}
              placeholder="0.00"
              placeholderTextColor={isDark ? COLORS.gray : COLORS.lightGray}
              value={amount}
              onChangeText={setAmount}
              keyboardType="numeric"
            />
          </View>
        </View>

        {/* Swap Button */}
        <View style={styles.swapButtonContainer}>
          <TouchableOpacity
            style={[styles.swapButton, { backgroundColor: COLORS.primary }]}
            onPress={() => { const tmp = fromChainId; setFromChainId(toChainId); setToChainId(tmp); setQuote(null); }}
          >
            <Text style={styles.swapIcon}>⇅</Text>
          </TouchableOpacity>
        </View>

        {/* To Chain */}
        <View style={[styles.chainCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.chainLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>To</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
            {chainOptions.map(chain => (
              <TouchableOpacity key={`to-${chain.id}`} style={[styles.chainChip, toChainId === chain.id && styles.chainChipSelected]} onPress={() => setToChainId(chain.id)}>
                <Text style={[styles.chainChipText, toChainId === chain.id && styles.chainChipTextSelected]}>{chain.name}</Text>
              </TouchableOpacity>
            ))}
          </ScrollView>
          <View style={[styles.receiveAmount, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
            <Text style={[styles.receiveLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>You will receive</Text>
            {loading ? (
              <ActivityIndicator color={COLORS.primary} style={{ marginTop: 4 }} />
            ) : (
              <Text style={[styles.receiveValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
                {quote?.outputAmount ?? quote?.toAmount ?? amount || '0.00'}
              </Text>
            )}
          </View>
        </View>

        {/* Bridge Info */}
        <View style={[styles.infoCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          {error && <Text style={[styles.infoValue, { color: '#ef4444', marginBottom: 8 }]}>{error}</Text>}
          <View style={styles.infoRow}>
            <Text style={[styles.infoLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Estimated Time</Text>
            <Text style={[styles.infoValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{estimatedTime}</Text>
          </View>
          <View style={styles.infoRow}>
            <Text style={[styles.infoLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Fee</Text>
            <Text style={[styles.infoValue, { color: COLORS.primary }]}>{quote?.fee ?? '—'}</Text>
          </View>
          <View style={styles.infoRow}>
            <Text style={[styles.infoLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Route</Text>
            <Text style={[styles.infoValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{quote?.bridge ?? quote?.bridgeName ?? quote?.route ?? 'Best available'}</Text>
          </View>
        </View>
      </ScrollView>

      {/* Bridge Button */}
      <View style={[styles.bottomBar, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <TouchableOpacity
          style={[styles.bridgeActionButton, { backgroundColor: COLORS.primary }, (executing || !quote) && { opacity: 0.5 }]}
          onPress={handleBridge}
          disabled={executing || !quote}
        >
          <Text style={styles.bridgeActionText}>{executing ? 'Bridging...' : 'Bridge'}</Text>
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  headerTitle: { fontSize: FONT_SIZES.xxl, fontWeight: 'bold' },
  content: { padding: SPACING.md, paddingBottom: 100 },
  chainCard: { padding: SPACING.md, borderRadius: 16 },
  chainLabel: { fontSize: FONT_SIZES.sm, fontWeight: '600', marginBottom: SPACING.sm },
  chainChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 20, marginRight: SPACING.sm, backgroundColor: COLORS.backgroundDark },
  chainChipSelected: { backgroundColor: COLORS.primary },
  chainChipText: { fontSize: FONT_SIZES.sm, color: COLORS.gray },
  chainChipTextSelected: { color: COLORS.white },
  amountContainer: { marginTop: SPACING.md },
  amountInput: { fontSize: 32, fontWeight: 'bold', padding: SPACING.md, borderRadius: 12 },
  receiveAmount: { marginTop: SPACING.md, padding: SPACING.md, borderRadius: 12 },
  receiveLabel: { fontSize: FONT_SIZES.sm },
  receiveValue: { fontSize: 24, fontWeight: 'bold', marginTop: 4 },
  swapButtonContainer: { alignItems: 'center', marginVertical: -20, zIndex: 1 },
  swapButton: { width: 44, height: 44, borderRadius: 22, justifyContent: 'center', alignItems: 'center', borderWidth: 4, borderColor: COLORS.backgroundDark },
  swapIcon: { fontSize: 20, color: COLORS.white },
  infoCard: { padding: SPACING.md, borderRadius: 12, marginTop: SPACING.sm },
  infoRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: SPACING.sm },
  infoLabel: { fontSize: FONT_SIZES.sm },
  infoValue: { fontSize: FONT_SIZES.sm, fontWeight: '600' },
  bottomBar: { position: 'absolute', bottom: 0, left: 0, right: 0, padding: SPACING.md, borderTopWidth: 1, borderTopColor: COLORS.borderDark },
  bridgeActionButton: { padding: SPACING.md, borderRadius: 12, alignItems: 'center' },
  bridgeActionText: { color: COLORS.white, fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
});

export default BridgeScreen;
