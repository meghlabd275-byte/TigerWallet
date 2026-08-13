/**
 * TigerWallet DApp Browser Screen - Complete Implementation
 * 
 * Built-in DApp browser with Web3 connectivity
 */

import React, { useEffect, useState } from 'react';
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
  ActivityIndicator,
  Linking,
} from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';
import { API } from '../../services/API';

interface DApp {
  id: string;
  name: string;
  category: string;
  icon: string;
  url: string;
  description: string;
}

const DAppBrowserScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('All');
  const [dapps, setDapps] = useState<DApp[]>([]);
  const [categories, setCategories] = useState<string[]>(['All']);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch the curated dApp directory from the canonical wallet_api
  // dapp_directory.go (~20 real protocol entries, no fabricated metrics).
  const loadDApps = async () => {
    setLoading(true);
    setError(null);
    try {
      const [dappsRes, catsRes] = await Promise.all([
        API.getDApps(),
        API.getDAppCategories().catch(() => null),
      ]);
      const list = dappsRes?.data?.dapps ?? dappsRes?.data ?? [];
      const mapped: DApp[] = (list as any[]).map((d) => ({
        id: d.id ?? d.name,
        name: d.name ?? 'dApp',
        category: d.category ?? 'Other',
        icon: d.logo ?? d.icon ?? '🔗',
        url: d.url ?? d.website ?? '',
        description: d.description ?? '',
      }));
      setDapps(mapped);
      const cats = catsRes?.data?.categories ?? catsRes?.data ?? [];
      if (Array.isArray(cats) && cats.length > 0) {
        setCategories(['All', ...cats]);
      } else {
        setCategories(['All', ...Array.from(new Set(mapped.map((d) => d.category)))]);
      }
    } catch (err) {
      setError('Failed to load dApp directory. Pull down to retry.');
      setDapps([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDApps();
  }, []);

  const filteredDApps = dapps.filter(dapp => {
    const matchesSearch = dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         dapp.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'All' || dapp.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const openDApp = (url: string) => {
    if (url) Linking.openURL(url).catch(() => {});
  };

  const renderDAppItem = ({ item }: { item: DApp }) => (
    <TouchableOpacity
      style={[styles.dappItem, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}
      onPress={() => openDApp(item.url)}
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
        ListEmptyComponent={
          <View style={styles.emptyContainer}>
            {loading ? (
              <ActivityIndicator color={COLORS.primary} />
            ) : (
              <Text style={{ color: isDark ? COLORS.gray : COLORS.lightGray, textAlign: 'center' }}>
                {error || 'No dApps found.'}
              </Text>
            )}
            {!loading && error && (
              <TouchableOpacity style={styles.retryButton} onPress={loadDApps}>
                <Text style={styles.retryButtonText}>Retry</Text>
              </TouchableOpacity>
            )}
          </View>
        }
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
  emptyContainer: { padding: SPACING.xl, alignItems: 'center' },
  retryButton: { marginTop: SPACING.md, backgroundColor: COLORS.primary, paddingHorizontal: 24, paddingVertical: 10, borderRadius: 8 },
  retryButtonText: { color: COLORS.white, fontWeight: '600' },
});

export default DAppBrowserScreen;
