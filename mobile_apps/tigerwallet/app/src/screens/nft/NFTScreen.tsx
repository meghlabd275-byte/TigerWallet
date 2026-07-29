/**
 * TigerWallet NFT Screen - Complete Implementation
 * 
 * NFT collection viewer and marketplace
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
  Image,
} from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';

interface NFT {
  id: string;
  name: string;
  collection: string;
  image: string;
  chain: string;
  price?: string;
  lastSale?: string;
}

const nfts: NFT[] = [
  { id: '1', name: 'Bored Ape #1234', collection: 'Bored Ape Yacht Club', image: '🦍', chain: 'Ethereum', price: '45 ETH', lastSale: '42 ETH' },
  { id: '2', name: 'Pudgy Penguin #567', collection: 'Pudgy Penguins', image: '🐧', chain: 'Ethereum', price: '3.5 ETH', lastSale: '3.2 ETH' },
  { id: '3', name: 'Azuki #890', collection: 'Azuki', image: '👘', chain: 'Ethereum', price: '12 ETH', lastSale: '15 ETH' },
  { id: '4', name: 'DeGod #321', collection: 'DeGods', image: '👽', chain: 'Solana', price: '250 SOL', lastSale: '280 SOL' },
  { id: '5', name: 'Moonbird #654', collection: 'Moonbirds', image: '🦉', chain: 'Ethereum', price: '4.2 ETH', lastSale: '5.5 ETH' },
  { id: '6', name: 'Clone X #987', collection: 'Clone X', image: '👾', chain: 'Solana', price: '8 ETH', lastSale: '9 ETH' },
];

const collections = ['All', 'BAYC', 'Pudgy', 'Azuki', 'DeGods', 'Moonbirds'];

const NFTScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [selectedCollection, setSelectedCollection] = useState('All');

  const filteredNFTs = selectedCollection === 'All' 
    ? nfts 
    : nfts.filter(n => n.collection.toLowerCase().includes(selectedCollection.toLowerCase().replace(' ', '').replace('yacht club', 'bored ape').replace('penguins', 'pudgy')));

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
            <Text style={styles.summaryStatValue}>6</Text>
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
});

export default NFTScreen;
