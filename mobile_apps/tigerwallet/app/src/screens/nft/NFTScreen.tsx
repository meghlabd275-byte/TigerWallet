/**
 * TigerWallet NFT Screen - Complete Implementation
 * 
 * NFT collection viewer and marketplace
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
  Image,
  ActivityIndicator,
} from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';
import { API } from '../../services/API';

interface NFT {
  id: string;
  name: string;
  collection: string;
  image: string;
  chain: string;
  price?: string;
  lastSale?: string;
}

const NFTScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const wallet = useSelector((state: RootState) => state.wallet.wallet);
  const selectedChainId = useSelector((state: RootState) => state.wallet.selectedChainId);
  const isDark = theme === 'dark';
  const [selectedCollection, setSelectedCollection] = useState('All');
  const [nfts, setNfts] = useState<NFT[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch the wallet's real NFT holdings from the canonical wallet_api
  // nft_service (on-chain ERC-721 reads). No hardcoded BAYC/Azuki mock data.
  const loadNFTs = async () => {
    if (!wallet?.id) {
      setNfts([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await API.getNFTs(wallet.id, selectedChainId);
      const list = res?.data?.nfts ?? res?.data?.tokens ?? res?.data ?? [];
      const mapped: NFT[] = (list as any[]).map((n) => ({
        id: n.id ?? n.token_id ?? n.tokenId ?? Math.random().toString(36),
        name: n.name ?? n.title ?? `${n.contract?.name ?? 'NFT'} #${n.token_id ?? n.tokenId ?? ''}`,
        collection: n.contract?.name ?? n.collection ?? n.contract_name ?? 'Collection',
        image: n.image ?? n.image_url ?? n.metadata?.image ?? '🖼️',
        chain: n.chain ?? 'Ethereum',
        price: n.price ?? n.last_price ?? undefined,
        lastSale: n.last_sale ?? n.last_sale_price ?? undefined,
      }));
      setNfts(mapped);
    } catch (err) {
      setError('Failed to load NFTs. Pull down to retry.');
      setNfts([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadNFTs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wallet?.id, selectedChainId]);

  // Build the collection filter dynamically from the real NFT data.
  const collections = ['All', ...Array.from(new Set(nfts.map((n) => n.collection).filter(Boolean)))];

  const filteredNFTs = selectedCollection === 'All'
    ? nfts
    : nfts.filter(n => n.collection === selectedCollection);

  const totalValue = nfts.reduce((sum, nft) => sum + (parseFloat(nft.price?.split(' ')[0] || '0')), 0);

  const renderNFT = ({ item }: { item: NFT }) => (
    <TouchableOpacity style={[styles.nftCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
      <View style={[styles.nftImage, { backgroundColor: COLORS.primary + '20' }]}>
        <Text style={styles.nftImageText}>{item.image}</Text>
      </View>
      <View style={styles.nftInfo}>
        <Text style={[styles.nftName, { color: isDark ? COLORS.textDark : COLORS.textLight }]} numberOfLines={1}>
          {item.name}
        </Text>
        <Text style={[styles.nftCollection, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
          {item.collection}
        </Text>
        <View style={styles.nftPriceRow}>
          <Text style={[styles.nftPrice, { color: COLORS.primary }]}>{item.price}</Text>
          <Text style={[styles.nftChain, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{item.chain}</Text>
        </View>
      </View>
    </TouchableOpacity>
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} backgroundColor={isDark ? COLORS.backgroundDark : COLORS.backgroundLight} />
      
      {/* Header */}
      <View style={styles.header}>
        <Text style={[styles.headerTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          NFTs
        </Text>
        <ThemeToggle />
      </View>

      {/* Portfolio Summary */}
      <View style={[styles.summaryCard, { backgroundColor: COLORS.accent }]}>
        <Text style={styles.summaryLabel}>Portfolio Value</Text>
        <Text style={styles.summaryValue}>${totalValue.toLocaleString()}</Text>
        <View style={styles.summaryRow}>
          <View style={styles.summaryStat}>
            <Text style={styles.summaryStatLabel}>Items</Text>
            <Text style={styles.summaryStatValue}>{nfts.length}</Text>
          </View>
          <View style={styles.summaryStat}>
            <Text style={styles.summaryStatLabel}>Collections</Text>
            <Text style={styles.summaryStatValue}>{collections.length - 1}</Text>
          </View>
        </View>
      </View>

      {/* Collection Filter */}
      <View style={styles.filterContainer}>
        <FlatList
          horizontal
          data={collections}
          renderItem={({ item }) => (
            <TouchableOpacity
              style={[styles.filterChip, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }, selectedCollection === item && styles.filterChipSelected]}
              onPress={() => setSelectedCollection(item)}
            >
              <Text style={[styles.filterChipText, { color: isDark ? COLORS.textDark : COLORS.textLight }, selectedCollection === item && styles.filterChipTextSelected]}>
                {item}
              </Text>
            </TouchableOpacity>
          )}
          keyExtractor={item => item}
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.filterList}
        />
      </View>

      {/* NFT Grid */}
      <FlatList
        data={filteredNFTs}
        renderItem={renderNFT}
        keyExtractor={item => item.id}
        numColumns={2}
        contentContainerStyle={styles.nftList}
        showsVerticalScrollIndicator={false}
        columnWrapperStyle={styles.nftRow}
        ListEmptyComponent={
          <View style={styles.emptyContainer}>
            {loading ? (
              <ActivityIndicator color={COLORS.primary} />
            ) : (
              <Text style={{ color: isDark ? COLORS.gray : COLORS.lightGray, textAlign: 'center' }}>
                {error || 'No NFTs found for this wallet.'}
              </Text>
            )}
            {!loading && error && (
              <TouchableOpacity style={styles.retryButton} onPress={loadNFTs}>
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
  filterContainer: { marginBottom: SPACING.sm },
  filterList: { paddingHorizontal: SPACING.md },
  filterChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 20, marginRight: SPACING.sm },
  filterChipSelected: { backgroundColor: COLORS.primary },
  filterChipText: { fontSize: FONT_SIZES.sm, fontWeight: '600' },
  filterChipTextSelected: { color: COLORS.white },
  nftList: { paddingHorizontal: SPACING.md, paddingBottom: SPACING.xl },
  nftRow: { justifyContent: 'space-between' },
  nftCard: { width: '48%', borderRadius: 12, marginBottom: SPACING.sm, overflow: 'hidden' },
  nftImage: { height: 150, justifyContent: 'center', alignItems: 'center' },
  nftImageText: { fontSize: 60 },
  nftInfo: { padding: SPACING.sm },
  nftName: { fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  nftCollection: { fontSize: FONT_SIZES.xs, marginTop: 2 },
  nftPriceRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: SPACING.sm },
  nftPrice: { fontSize: FONT_SIZES.sm, fontWeight: 'bold' },
  nftChain: { fontSize: FONT_SIZES.xs },
  emptyContainer: { padding: SPACING.xl, alignItems: 'center' },
  retryButton: { marginTop: SPACING.md, backgroundColor: COLORS.primary, paddingHorizontal: 24, paddingVertical: 10, borderRadius: 8 },
  retryButtonText: { color: COLORS.white, fontWeight: '600' },
});

export default NFTScreen;
