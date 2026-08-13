import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, FlatList, ActivityIndicator } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../store';
import { API } from '../services/API';
import { useThemeStore } from '../stores/ThemeStore';
import { useNavigation } from '@react-navigation/native';

interface TxItem {
  hash: string;
  from: string;
  to: string;
  value: string;
  timestamp: number;
  status?: string;
}

const WalletDetailsScreen: React.FC = () => {
  const { theme } = useThemeStore();
  const navigation = useNavigation();
  const wallet = useSelector((state: RootState) => state.wallet.wallet);
  const selectedChainId = useSelector((state: RootState) => state.wallet.selectedChainId);

  const address = wallet?.addresses?.[selectedChainId] || 'No address for this chain';
  const [txs, setTxs] = useState<TxItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch the real transaction history for the wallet's address on the
  // selected chain from the canonical wallet_api (no hardcoded address).
  const loadTxs = async () => {
    if (!wallet?.id) {
      setTxs([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await API.getTransactions(wallet.id, selectedChainId);
      const list = res?.data?.transactions ?? res?.data?.txs ?? res?.data ?? [];
      const mapped: TxItem[] = (list as any[]).map((t) => ({
        hash: t.hash ?? t.tx_hash ?? t.transactionHash ?? '',
        from: t.from ?? '',
        to: t.to ?? '',
        value: t.value ?? t.amount ?? '0',
        timestamp: t.timestamp ?? t.timeStamp ?? t.time ?? 0,
        status: t.status,
      }));
      setTxs(mapped);
    } catch (err) {
      setError('Failed to load transactions.');
      setTxs([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTxs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wallet?.id, selectedChainId]);

  const renderTx = ({ item }: { item: TxItem }) => (
    <View style={[styles.txCard, { backgroundColor: theme.colors.surface }]}>
      <Text style={[styles.txHash, { color: theme.colors.textSecondary }]} numberOfLines={1}>{item.hash || '—'}</Text>
      <View style={styles.txRow}>
        <Text style={[styles.txLabel, { color: theme.colors.textTertiary }]}>From</Text>
        <Text style={[styles.txValue, { color: theme.colors.text }]} numberOfLines={1}>{item.from || '—'}</Text>
      </View>
      <View style={styles.txRow}>
        <Text style={[styles.txLabel, { color: theme.colors.textTertiary }]}>To</Text>
        <Text style={[styles.txValue, { color: theme.colors.text }]} numberOfLines={1}>{item.to || '—'}</Text>
      </View>
      <View style={styles.txRow}>
        <Text style={[styles.txLabel, { color: theme.colors.textTertiary }]}>Value</Text>
        <Text style={[styles.txValue, { color: theme.colors.primary }]}>{item.value}</Text>
      </View>
    </View>
  );

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}><Text style={[styles.backButton, { color: theme.colors.primary }]}>← Back</Text></TouchableOpacity>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>Wallet Details</Text>
        <View style={{ width: 50 }} />
      </View>
      <ScrollView style={styles.content}>
        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>Address</Text>
          <Text style={[styles.address, { color: theme.colors.text }]}>{address}</Text>
        </View>
        <Text style={[styles.sectionTitle, { color: theme.colors.text }]}>Transactions</Text>
        {loading ? (
          <ActivityIndicator color={theme.colors.primary} style={{ padding: 20 }} />
        ) : error ? (
          <Text style={[styles.emptyText, { color: theme.colors.textTertiary }]}>{error}</Text>
        ) : (
          <FlatList
            data={txs}
            renderItem={renderTx}
            keyExtractor={(item, idx) => item.hash || `${idx}`}
            scrollEnabled={false}
            ListEmptyComponent={<Text style={[styles.emptyText, { color: theme.colors.textTertiary }]}>No transactions found.</Text>}
          />
        )}
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
  label: { fontSize: 12, marginBottom: 8 },
  address: { fontSize: 14, fontFamily: 'monospace' },
  sectionTitle: { fontSize: 16, fontWeight: '700', marginBottom: 12 },
  txCard: { padding: 12, borderRadius: 10, marginBottom: 10 },
  txHash: { fontSize: 11, fontFamily: 'monospace', marginBottom: 6 },
  txRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 },
  txLabel: { fontSize: 12 },
  txValue: { fontSize: 12, flex: 1, marginLeft: 12, textAlign: 'right' },
  emptyText: { padding: 20, textAlign: 'center' },
});
export default WalletDetailsScreen;
