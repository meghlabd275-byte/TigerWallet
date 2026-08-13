// Swap Screen — real quote/execute via the canonical wallet_api swap endpoints.
import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, TextInput, ActivityIndicator, Alert } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../store';
import { API } from '../services/API';
import { useThemeStore } from '../stores/ThemeStore';
import { useNavigation } from '@react-navigation/native';

const SwapScreen: React.FC = () => {
  const { theme } = useThemeStore();
  const navigation = useNavigation();
  const wallet = useSelector((state: RootState) => state.wallet.wallet);
  const selectedChainId = useSelector((state: RootState) => state.wallet.selectedChainId);

  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDT');
  const [amount, setAmount] = useState('');
  const [quote, setQuote] = useState<any>(null);
  const [loadingQuote, setLoadingQuote] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch a real swap quote from the backend (debounced via effect on amount).
  useEffect(() => {
    if (!amount || parseFloat(amount) <= 0) {
      setQuote(null);
      setError(null);
      return;
    }
    let cancelled = false;
    setLoadingQuote(true);
    setError(null);
    const timer = setTimeout(async () => {
      try {
        const res = await API.getSwapQuote({
          fromChainId: selectedChainId,
          toChainId: selectedChainId,
          fromToken,
          toToken,
          amount,
          fromAddress: wallet?.addresses?.[selectedChainId] || '',
        });
        if (!cancelled) {
          if (res?.success === false) {
            setError(res.error || 'No route found for this pair');
            setQuote(null);
          } else {
            setQuote(res?.data);
          }
        }
      } catch (err) {
        if (!cancelled) setError('Failed to fetch quote');
      } finally {
        if (!cancelled) setLoadingQuote(false);
      }
    }, 400);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [amount, fromToken, toToken, selectedChainId, wallet]);

  const handleSwap = async () => {
    if (!wallet?.id || !quote) {
      Alert.alert('Error', 'Enter an amount and fetch a quote first');
      return;
    }
    setExecuting(true);
    try {
      const res = await API.executeSwap({
        quoteId: quote?.quoteId ?? quote?.quote_id ?? quote?.id ?? '',
        walletId: wallet.id,
        slippage: 0.5,
      });
      if (res?.success === false) {
        Alert.alert('Error', res.error || 'Swap failed');
      } else {
        Alert.alert('Success', 'Swap submitted!');
        setAmount('');
        setQuote(null);
      }
    } catch (err) {
      Alert.alert('Error', 'Swap request failed');
    } finally {
      setExecuting(false);
    }
  };

  const swapTokens = () => {
    setFromToken(toToken);
    setToToken(fromToken);
    setQuote(null);
  };

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}>
          <Text style={[styles.backButton, { color: theme.colors.primary }]}>← Back</Text>
        </TouchableOpacity>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>Swap</Text>
        <View style={{ width: 50 }} />
      </View>
      <ScrollView style={styles.content}>
        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>From</Text>
          <View style={styles.tokenRow}>
            <TextInput
              style={[styles.tokenSymbol, { color: theme.colors.text }]}
              value={fromToken}
              onChangeText={setFromToken}
              autoCapitalize="characters"
            />
          </View>
          <View style={[styles.input, { backgroundColor: theme.colors.surfaceVariant }]}>
            <TextInput
              style={[styles.inputText, { color: theme.colors.text }]}
              value={amount}
              onChangeText={setAmount}
              placeholder="0.00"
              keyboardType="decimal-pad"
              placeholderTextColor={theme.colors.textTertiary}
            />
          </View>
        </View>

        <TouchableOpacity style={[styles.swapButton, { backgroundColor: theme.colors.primary }]} onPress={swapTokens}>
          <Text style={styles.swapIcon}>⇅</Text>
        </TouchableOpacity>

        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>To</Text>
          <View style={styles.tokenRow}>
            <TextInput
              style={[styles.tokenSymbol, { color: theme.colors.text }]}
              value={toToken}
              onChangeText={setToToken}
              autoCapitalize="characters"
            />
          </View>
          <View style={[styles.input, { backgroundColor: theme.colors.surfaceVariant }]}>
            <Text style={[styles.inputText, { color: theme.colors.text }]}>
              {quote?.toAmount ?? quote?.outputAmount ?? quote?.to_amount ?? '0.00'}
            </Text>
          </View>
        </View>

        <View style={[styles.rateInfo, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.rateLabel, { color: theme.colors.textSecondary }]}>Rate</Text>
          {loadingQuote && <ActivityIndicator color={theme.colors.primary} size="small" />}
          {error && <Text style={[styles.rateValue, { color: '#ef4444' }]}>{error}</Text>}
          {!loadingQuote && !error && quote && (
            <Text style={[styles.rateValue, { color: theme.colors.text }]}>
              {`1 ${fromToken} = ${quote?.rate ?? quote?.price ?? '—'} ${toToken}`}
            </Text>
          )}
          {!loadingQuote && !error && !quote && (
            <Text style={[styles.rateValue, { color: theme.colors.textTertiary }]}>Enter an amount</Text>
          )}
        </View>

        <TouchableOpacity
          style={[styles.swapButtonFull, { backgroundColor: theme.colors.primary }, (executing || !quote) && { opacity: 0.5 }]}
          onPress={handleSwap}
          disabled={executing || !quote}
        >
          <Text style={styles.swapButtonText}>{executing ? 'Swapping...' : 'Swap'}</Text>
        </TouchableOpacity>
      </ScrollView>
    </View>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingTop: 50, paddingHorizontal: 20, paddingBottom: 20 },
  backButton: { fontSize: 16, fontWeight: '600' },
  headerTitle: { fontSize: 18, fontWeight: '600' },
  content: { flex: 1, padding: 20 },
  card: { padding: 16, borderRadius: 12, marginBottom: 16 },
  label: { fontSize: 12, fontWeight: '500', marginBottom: 12 },
  tokenRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 12 },
  tokenSymbol: { fontSize: 18, fontWeight: '600' },
  balance: { fontSize: 14 },
  input: { padding: 16, borderRadius: 12 },
  inputText: { fontSize: 24, fontWeight: '700' },
  swapButton: { width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center', alignSelf: 'center', marginVertical: -12, zIndex: 10 },
  swapIcon: { fontSize: 20, color: '#FFF' },
  rateInfo: { padding: 16, borderRadius: 12, marginBottom: 24 },
  rateLabel: { fontSize: 12, marginBottom: 4 },
  rateValue: { fontSize: 14, fontWeight: '500' },
  swapButtonFull: { padding: 18, borderRadius: 12, alignItems: 'center' },
  swapButtonText: { color: '#FFF', fontSize: 18, fontWeight: '700' },
});

export default SwapScreen;
