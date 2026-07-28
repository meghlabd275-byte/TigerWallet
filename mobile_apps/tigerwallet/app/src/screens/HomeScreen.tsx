// ============================================================================
// TigerWallet - Home Screen
// Main Dashboard with Portfolio Overview
// ============================================================================

import React, { useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  RefreshControl,
  Dimensions,
  StatusBar,
} from 'react-native';
import { useThemeStore } from '../stores/ThemeStore';
import { walletService } from '../services/WalletService';
import { blockchainService } from '../services/BlockchainService';
import { Wallet, TokenBalance } from '../types/wallet';
import { cryptoService } from '../services/CryptoService';
import { useNavigation } from '@react-navigation/native';

const { width } = Dimensions.get('window');

interface Props {
  navigation: any;
}

const HomeScreen: React.FC<Props> = ({ navigation }) => {
  const { theme, isDark, toggleTheme } = useThemeStore();
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [activeWallet, setActiveWallet] = useState<Wallet | null>(null);
  const [totalValue, setTotalValue] = useState(0);
  const [refreshing, setRefreshing] = useState(false);
  const [selectedChain, setSelectedChain] = useState(1);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const allWallets = walletService.getWallets();
      setWallets(allWallets);
      
      const active = walletService.getActiveWallet();
      setActiveWallet(active || null);

      if (active) {
        const value = await walletService.getPortfolioValue(active.id);
        setTotalValue(value);
      }
    } catch (error) {
      console.error('Failed to load data:', error);
    }
  };

  const onRefresh = async () => {
    setRefreshing(true);
    await loadData();
    setRefreshing(false);
  };

  const renderHeader = () => (
    <View style={[styles.header, { backgroundColor: theme.colors.surface }]}>
      <View style={styles.headerTop}>
        <TouchableOpacity 
          style={[styles.menuButton, { backgroundColor: theme.colors.surfaceVariant }]}
          onPress={() => navigation.openDrawer?.()}
        >
          <Text style={[styles.menuIcon, { color: theme.colors.text }]}>☰</Text>
        </TouchableOpacity>
        
        <View style={styles.headerRight}>
          <TouchableOpacity 
            style={[styles.iconButton, { backgroundColor: theme.colors.surfaceVariant }]}
            onPress={toggleTheme}
          >
            <Text style={styles.iconText}>{isDark ? '☀️' : '🌙'}</Text>
          </TouchableOpacity>
          
          <TouchableOpacity 
            style={[styles.iconButton, { backgroundColor: theme.colors.surfaceVariant }]}
            onPress={() => navigation.navigate('Notifications')}
          >
            <Text style={styles.iconText}>🔔</Text>
          </TouchableOpacity>
        </View>
      </View>

      <View style={styles.balanceContainer}>
        <Text style={[styles.balanceLabel, { color: theme.colors.textSecondary }]}>
          Total Balance
        </Text>
        <Text style={[styles.balanceValue, { color: theme.colors.text }]}>
          ${totalValue.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
        </Text>
        
        <View style={styles.changeContainer}>
          <Text style={[styles.changePositive, { color: theme.colors.positive }]}>
            ↑ $0.00 (0%)
          </Text>
          <Text style={[styles.changeLabel, { color: theme.colors.textTertiary }]}>
            24h change
          </Text>
        </View>
      </View>

      <View style={styles.actionButtons}>
        <TouchableOpacity 
          style={[styles.actionButton, { backgroundColor: theme.colors.primary }]}
          onPress={() => navigation.navigate('Send')}
        >
          <Text style={styles.actionIcon}>↑</Text>
          <Text style={[styles.actionText, { color: '#FFFFFF' }]}>Send</Text>
        </TouchableOpacity>
        
        <TouchableOpacity 
          style={[styles.actionButton, { backgroundColor: theme.colors.primary }]}
          onPress={() => navigation.navigate('Receive')}
        >
          <Text style={styles.actionIcon}>↓</Text>
          <Text style={[styles.actionText, { color: '#FFFFFF' }]}>Receive</Text>
        </TouchableOpacity>
        
        <TouchableOpacity 
          style={[styles.actionButton, { backgroundColor: theme.colors.primary }]}
          onPress={() => navigation.navigate('Swap')}
        >
          <Text style={styles.actionIcon}>⇄</Text>
          <Text style={[styles.actionText, { color: '#FFFFFF' }]}>Swap</Text>
        </TouchableOpacity>
        
        <TouchableOpacity 
          style={[styles.actionButton, { backgroundColor: theme.colors.primary }]}
          onPress={() => navigation.navigate('Buy')}
        >
          <Text style={styles.actionIcon}>💳</Text>
          <Text style={[styles.actionText, { color: '#FFFFFF' }]}>Buy</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  const renderChainSelector = () => (
    <ScrollView 
      horizontal 
      showsHorizontalScrollIndicator={false}
      style={styles.chainSelector}
      contentContainerStyle={styles.chainSelectorContent}
    >
      {blockchainService.getSupportedChains().slice(0, 10).map((chain) => (
        <TouchableOpacity
          key={chain.id}
          style={[
            styles.chainItem,
            { 
              backgroundColor: selectedChain === chain.id 
                ? theme.colors.primary 
                : theme.colors.surfaceVariant 
            }
          ]}
          onPress={() => setSelectedChain(chain.id)}
        >
          <Text style={styles.chainIcon}>{chain.symbol[0]}</Text>
          <Text style={[
            styles.chainName,
            { color: selectedChain === chain.id ? '#FFFFFF' : theme.colors.text }
          ]}>
            {chain.symbol}
          </Text>
        </TouchableOpacity>
      ))}
    </ScrollView>
  );

  const renderWalletCard = (wallet: Wallet) => {
    const address = wallet.addresses[selectedChain] || Object.values(wallet.addresses)[0];
    
    return (
      <TouchableOpacity 
        key={wallet.id}
        style={[styles.walletCard, { backgroundColor: theme.colors.surface }]}
        onPress={() => navigation.navigate('WalletDetails', { walletId: wallet.id })}
      >
        <View style={styles.walletHeader}>
          <View style={[styles.walletIcon, { backgroundColor: theme.colors.primary }]}>
            <Text style={styles.walletIconText}>
              {wallet.name.charAt(0).toUpperCase()}
            </Text>
          </View>
          <View style={styles.walletInfo}>
            <Text style={[styles.walletName, { color: theme.colors.text }]}>
              {wallet.name}
            </Text>
            <Text style={[styles.walletAddress, { color: theme.colors.textSecondary }]}>
              {blockchainService.formatAddress(address)}
            </Text>
          </View>
          <View style={styles.walletBadge}>
            {wallet.isBackedUp && (
              <Text style={styles.badgeText}>✓</Text>
            )}
          </View>
        </View>
      </TouchableOpacity>
    );
  };

  const renderQuickActions = () => (
    <View style={styles.quickActions}>
      <TouchableOpacity 
        style={[styles.quickAction, { backgroundColor: theme.colors.surface }]}
        onPress={() => navigation.navigate('Bridge')}
      >
        <Text style={styles.quickIcon}>🌉</Text>
        <Text style={[styles.quickLabel, { color: theme.colors.text }]}>Bridge</Text>
      </TouchableOpacity>
      
      <TouchableOpacity 
        style={[styles.quickAction, { backgroundColor: theme.colors.surface }]}
        onPress={() => navigation.navigate('Staking')}
      >
        <Text style={styles.quickIcon}>📈</Text>
        <Text style={[styles.quickLabel, { color: theme.colors.text }]}>Stake</Text>
      </TouchableOpacity>
      
      <TouchableOpacity 
        style={[styles.quickAction, { backgroundColor: theme.colors.surface }]}
        onPress={() => navigation.navigate('NFTs')}
      >
        <Text style={styles.quickIcon}>🖼️</Text>
        <Text style={[styles.quickLabel, { color: theme.colors.text }]}>NFTs</Text>
      </TouchableOpacity>
      
      <TouchableOpacity 
        style={[styles.quickAction, { backgroundColor: theme.colors.surface }]}
        onPress={() => navigation.navigate('DApps')}
      >
        <Text style={styles.quickIcon}>🌐</Text>
        <Text style={[styles.quickLabel, { color: theme.colors.text }]}>DApps</Text>
      </TouchableOpacity>
      
      <TouchableOpacity 
        style={[styles.quickAction, { backgroundColor: theme.colors.surface }]}
        onPress={() => navigation.navigate('History')}
      >
        <Text style={styles.quickIcon}>📜</Text>
        <Text style={[styles.quickLabel, { color: theme.colors.text }]}>History</Text>
      </TouchableOpacity>
    </View>
  );

  const renderRecentTransactions = () => (
    <View style={styles.transactionsSection}>
      <View style={styles.sectionHeader}>
        <Text style={[styles.sectionTitle, { color: theme.colors.text }]}>
          Recent Transactions
        </Text>
        <TouchableOpacity onPress={() => navigation.navigate('History')}>
          <Text style={[styles.seeAll, { color: theme.colors.primary }]}>
            See All
          </Text>
        </TouchableOpacity>
      </View>
      
      <View style={[styles.emptyState, { backgroundColor: theme.colors.surface }]}>
        <Text style={styles.emptyIcon}>📭</Text>
        <Text style={[styles.emptyText, { color: theme.colors.textSecondary }]}>
          No recent transactions
        </Text>
        <Text style={[styles.emptySubtext, { color: theme.colors.textTertiary }]}>
          Your transaction history will appear here
        </Text>
      </View>
    </View>
  );

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <StatusBar 
        barStyle={isDark ? 'light-content' : 'dark-content'}
        backgroundColor={theme.colors.background}
      />
      
      <ScrollView
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={onRefresh}
            tintColor={theme.colors.primary}
          />
        }
        showsVerticalScrollIndicator={false}
      >
        {renderHeader()}
        {renderChainSelector()}
        
        <View style={styles.content}>
          <View style={styles.walletsSection}>
            <View style={styles.sectionHeader}>
              <Text style={[styles.sectionTitle, { color: theme.colors.text }]}>
                My Wallets
              </Text>
              <TouchableOpacity onPress={() => navigation.navigate('AddWallet')}>
                <Text style={[styles.addButton, { color: theme.colors.primary }]}>
                  + Add
                </Text>
              </TouchableOpacity>
            </View>
            {wallets.map(renderWalletCard)}
          </View>

          {renderQuickActions()}
          {renderRecentTransactions()}
        </View>
      </ScrollView>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    paddingTop: 50,
    paddingHorizontal: 20,
    paddingBottom: 20,
    borderBottomLeftRadius: 24,
    borderBottomRightRadius: 24,
  },
  headerTop: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 20,
  },
  menuButton: {
    width: 44,
    height: 44,
    borderRadius: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  menuIcon: {
    fontSize: 20,
  },
  headerRight: {
    flexDirection: 'row',
    gap: 10,
  },
  iconButton: {
    width: 44,
    height: 44,
    borderRadius: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  iconText: {
    fontSize: 18,
  },
  balanceContainer: {
    alignItems: 'center',
    marginVertical: 20,
  },
  balanceLabel: {
    fontSize: 14,
    marginBottom: 8,
  },
  balanceValue: {
    fontSize: 42,
    fontWeight: '700',
  },
  changeContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 8,
    gap: 8,
  },
  changePositive: {
    fontSize: 14,
    fontWeight: '600',
  },
  changeLabel: {
    fontSize: 12,
  },
  actionButtons: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    marginTop: 24,
  },
  actionButton: {
    alignItems: 'center',
    padding: 12,
    borderRadius: 12,
    minWidth: 70,
  },
  actionIcon: {
    fontSize: 20,
    marginBottom: 4,
  },
  actionText: {
    fontSize: 12,
    fontWeight: '600',
  },
  chainSelector: {
    marginTop: 16,
  },
  chainSelectorContent: {
    paddingHorizontal: 20,
    gap: 10,
  },
  chainItem: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 20,
    marginRight: 10,
    gap: 6,
  },
  chainIcon: {
    fontSize: 16,
    fontWeight: '700',
    color: '#FFFFFF',
  },
  chainName: {
    fontSize: 12,
    fontWeight: '600',
  },
  content: {
    padding: 20,
  },
  walletsSection: {
    marginBottom: 20,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
  },
  addButton: {
    fontSize: 14,
    fontWeight: '600',
  },
  seeAll: {
    fontSize: 14,
    fontWeight: '500',
  },
  walletCard: {
    padding: 16,
    borderRadius: 16,
    marginBottom: 12,
  },
  walletHeader: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  walletIcon: {
    width: 44,
    height: 44,
    borderRadius: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  walletIconText: {
    color: '#FFFFFF',
    fontSize: 18,
    fontWeight: '700',
  },
  walletInfo: {
    flex: 1,
    marginLeft: 12,
  },
  walletName: {
    fontSize: 16,
    fontWeight: '600',
  },
  walletAddress: {
    fontSize: 12,
    marginTop: 2,
  },
  walletBadge: {
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: '#28A745',
    justifyContent: 'center',
    alignItems: 'center',
  },
  badgeText: {
    color: '#FFFFFF',
    fontSize: 12,
  },
  quickActions: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
    marginBottom: 20,
  },
  quickAction: {
    width: '18%',
    aspectRatio: 1,
    borderRadius: 12,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 10,
  },
  quickIcon: {
    fontSize: 24,
    marginBottom: 4,
  },
  quickLabel: {
    fontSize: 10,
    fontWeight: '500',
  },
  transactionsSection: {
    marginBottom: 40,
  },
  emptyState: {
    padding: 40,
    borderRadius: 16,
    alignItems: 'center',
  },
  emptyIcon: {
    fontSize: 40,
    marginBottom: 12,
  },
  emptyText: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 4,
  },
  emptySubtext: {
    fontSize: 12,
    textAlign: 'center',
  },
});

export default HomeScreen;
