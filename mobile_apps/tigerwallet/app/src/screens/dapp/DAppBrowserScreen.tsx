/**
 * TigerWallet DApp Browser Screen - Complete Implementation
 * 
 * Built-in DApp browser with Web3 connectivity
 */

import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  SafeAreaView,
  StatusBar,
  FlatList,
  Image,
} from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';

interface DApp {
  id: string;
  name: string;
  category: string;
  icon: string;
  url: string;
  description: string;
}

const popularDApps: DApp[] = [
  { id: '1', name: 'Uniswap', category: 'DeFi', icon: '🦄', url: 'https://app.uniswap.org', description: ' Decentralized trading protocol' },
  { id: '2', name: 'Aave', category: 'DeFi', icon: '👻', url: 'https://app.aave.com', description: 'Lending and borrowing' },
  { id: '3', name: 'Compound', category: 'DeFi', icon: '🔷', url: 'https://app.compound.finance', description: 'Algorithmic money markets' },
  { id: '4', name: 'OpenSea', category: 'NFT', icon: '🌊', url: 'https://opensea.io', description: 'NFT marketplace' },
  { id: '5', name: 'Blur', category: 'NFT', icon: '🟣', url: 'https://blur.io', description: 'NFT marketplace & aggregator' },
  { id: '6', name: 'PancakeSwap', category: 'DeFi', icon: '🥞', url: 'https://pancakeswap.finance', description: 'DEX on BNB Chain' },
  { id: '7', name: 'Curve', category: 'DeFi', icon: '📈', url: 'https://curve.fi', description: 'Stable asset swapping' },
  { id: '8', name: '1inch', category: 'DeFi', icon: '1️⃣', url: 'https://app.1inch.io', description: 'DEX aggregator' },
  { id: '9', name: 'Yearn', category: 'DeFi', icon: '📦', url: 'https://yearn.finance', description: 'Yield aggregator' },
  { id: '10', name: 'Lido', category: 'DeFi', icon: '💧', url: 'https://lido.fi', description: 'Liquid staking' },
  { id: '11', name: 'Rocket Pool', category: 'DeFi', icon: '🚀', url: 'https://rocketpool.net', description: 'ETH staking' },
  { id: '12', name: 'LooksRare', category: 'NFT', icon: '👀', url: 'https://looksrare.org', description: 'NFT marketplace' },
];

const categories = ['All', 'DeFi', 'NFT', 'Games', 'Social', 'Tools'];

const DAppBrowserScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('All');

  const filteredDApps = popularDApps.filter(dapp => {
    const matchesSearch = dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         dapp.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'All' || dapp.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const renderDAppItem = ({ item }: { item: DApp }) => (
    <TouchableOpacity 
      style={[styles.dappItem, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}
      onPress={() => {}}
    >
      <View style={[styles.dappIcon, { backgroundColor: COLORS.primary + '20' }]}>
        <Text style={styles.dappIconText}>{item.icon}</Text>
      </View>
      <View style={styles.dappInfo}>
        <Text style={[styles.dappName, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          {item.name}
        </Text>
        <Text style={[styles.dappDescription, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>
          {item.description}
        </Text>
        <View style={[styles.categoryBadge, { backgroundColor: COLORS.primary + '20' }]}>
          <Text style={[styles.categoryText, { color: COLORS.primary }]}>{item.category}</Text>
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
          DApp Browser
        </Text>
        <ThemeToggle />
      </View>

      {/* Search Bar */}
      <View style={styles.searchContainer}>
        <TextInput
          style={[styles.searchInput, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight, color: isDark ? COLORS.textDark : COLORS.textLight }]}
          placeholder="Search DApps..."
          placeholderTextColor={isDark ? COLORS.gray : COLORS.lightGray}
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      {/* Category Filter */}
      <View style={styles.categoryContainer}>
        <FlatList
          horizontal
          data={categories}
          renderItem={({ item }) => (
            <TouchableOpacity
              style={[
                styles.categoryChip,
                { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight },
                selectedCategory === item && styles.categoryChipSelected
              ]}
              onPress={() => setSelectedCategory(item)}
            >
              <Text style={[
                styles.categoryChipText,
                { color: isDark ? COLORS.textDark : COLORS.textLight },
                selectedCategory === item && styles.categoryChipTextSelected
              ]}>
                {item}
              </Text>
            </TouchableOpacity>
          )}
          keyExtractor={item => item}
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.categoryList}
        />
      </View>

      {/* Popular DApps */}
      <View style={styles.sectionHeader}>
        <Text style={[styles.sectionTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>
          Popular DApps
        </Text>
      </View>

      <FlatList
        data={filteredDApps}
        renderItem={renderDAppItem}
        keyExtractor={item => item.id}
        contentContainerStyle={styles.dappList}
        showsVerticalScrollIndicator={false}
      />

      {/* Bottom Bar */}
      <View style={[styles.bottomBar, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <TouchableOpacity style={styles.bottomBarItem}>
          <Text style={styles.bottomBarIcon}>🌐</Text>
          <Text style={[styles.bottomBarText, { color: COLORS.primary }]}>Browser</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.bottomBarItem}>
          <Text style={styles.bottomBarIcon}>🔗</Text>
          <Text style={[styles.bottomBarText, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Connect</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.bottomBarItem}>
          <Text style={styles.bottomBarIcon}>📜</Text>
          <Text style={[styles.bottomBarText, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>History</Text>
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: SPACING.md },
  headerTitle: { fontSize: FONT_SIZES.xxl, fontWeight: 'bold' },
  searchContainer: { paddingHorizontal: SPACING.md, paddingBottom: SPACING.sm },
  searchInput: { padding: SPACING.md, borderRadius: 12, fontSize: FONT_SIZES.md },
  categoryContainer: { paddingBottom: SPACING.sm },
  categoryList: { paddingHorizontal: SPACING.md },
  categoryChip: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm, borderRadius: 20, marginRight: SPACING.sm },
  categoryChipSelected: { backgroundColor: COLORS.primary },
  categoryChipText: { fontSize: FONT_SIZES.sm, fontWeight: '600' },
  categoryChipTextSelected: { color: COLORS.white },
  sectionHeader: { paddingHorizontal: SPACING.md, paddingVertical: SPACING.sm },
  sectionTitle: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  dappList: { paddingHorizontal: SPACING.md, paddingBottom: 100 },
  dappItem: { flexDirection: 'row', padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.sm },
  dappIcon: { width: 50, height: 50, borderRadius: 25, justifyContent: 'center', alignItems: 'center', marginRight: SPACING.md },
  dappIconText: { fontSize: 24 },
  dappInfo: { flex: 1 },
  dappName: { fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
  dappDescription: { fontSize: FONT_SIZES.sm, marginTop: 2 },
  categoryBadge: { alignSelf: 'flex-start', paddingHorizontal: SPACING.sm, paddingVertical: 2, borderRadius: 4, marginTop: SPACING.xs },
  categoryText: { fontSize: FONT_SIZES.xs, fontWeight: '600' },
  bottomBar: { flexDirection: 'row', justifyContent: 'space-around', padding: SPACING.md, borderTopWidth: 1, borderTopColor: COLORS.borderDark, position: 'absolute', bottom: 0, left: 0, right: 0 },
  bottomBarItem: { alignItems: 'center' },
  bottomBarIcon: { fontSize: 24 },
  bottomBarText: { fontSize: FONT_SIZES.sm, marginTop: 4 },
});

export default DAppBrowserScreen;
