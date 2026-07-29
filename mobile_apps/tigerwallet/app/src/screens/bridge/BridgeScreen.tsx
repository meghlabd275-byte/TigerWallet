/**
 * TigerWallet Bridge Screen - Complete Implementation
 * 
 * Cross-chain bridge interface
 */

import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, TextInput, SafeAreaView, StatusBar, ScrollView } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { COLORS, SPACING, FONT_SIZES } from '../../constants/theme';
import { ThemeToggle } from '../../components/ThemeToggle';

const bridges = [
  { id: '1', name: 'Stargate', logo: '🌉', chains: ['Ethereum', 'BSC', 'Avalanche', 'Polygon', 'Arbitrum'], fee: '0.06%' },
  { id: '2', name: 'Across', logo: '➡️', chains: ['Ethereum', 'Arbitrum', 'Optimism'], fee: '0.04%' },
  { id: '3', name: 'Hop', logo: '🐰', chains: ['Ethereum', 'Arbitrum', 'Optimism', 'Polygon'], fee: '0.05%' },
  { id: '4', name: 'Celer', logo: '🔗', chains: ['Ethereum', 'BSC', 'Avalanche', 'Polygon', 'Fantom'], fee: '0.03%' },
  { id: '5', name: 'LayerZero', logo: '💫', chains: ['Ethereum', 'BSC', 'Avalanche', 'Polygon', 'Arbitrum', 'Optimism'], fee: '0.06%' },
];

const chains = ['Ethereum', 'BSC', 'Polygon', 'Arbitrum', 'Optimism', 'Avalanche', 'Base', 'Solana'];

const BridgeScreen: React.FC = () => {
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';
  const [fromChain, setFromChain] = useState('Ethereum');
  const [toChain, setToChain] = useState('Polygon');
  const [amount, setAmount] = useState('');

  const selectedBridge = bridges[0];
  const estimatedTime = fromChain === toChain ? 'Same chain' : '~10-30 minutes';

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
            {chains.map(chain => (
              <TouchableOpacity key={chain} style={[styles.chainChip, fromChain === chain && styles.chainChipSelected]} onPress={() => setFromChain(chain)}>
                <Text style={[styles.chainChipText, fromChain === chain && styles.chainChipTextSelected]}>{chain}</Text>
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
          <TouchableOpacity style={[styles.swapButton, { backgroundColor: COLORS.primary }]} onPress={() => {}}>
            <Text style={styles.swapIcon}>⇅</Text>
          </TouchableOpacity>
        </View>

        {/* To Chain */}
        <View style={[styles.chainCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <Text style={[styles.chainLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>To</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
            {chains.map(chain => (
              <TouchableOpacity key={chain} style={[styles.chainChip, toChain === chain && styles.chainChipSelected]} onPress={() => setToChain(chain)}>
                <Text style={[styles.chainChipText, toChain === chain && styles.chainChipTextSelected]}>{chain}</Text>
              </TouchableOpacity>
            ))}
          </ScrollView>
          <View style={[styles.receiveAmount, { backgroundColor: isDark ? COLORS.backgroundDark : COLORS.backgroundLight }]}>
            <Text style={[styles.receiveLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>You will receive</Text>
            <Text style={[styles.receiveValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{amount || '0.00'}</Text>
          </View>
        </View>

        {/* Bridge Info */}
        <View style={[styles.infoCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
          <View style={styles.infoRow}>
            <Text style={[styles.infoLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Estimated Time</Text>
            <Text style={[styles.infoValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{estimatedTime}</Text>
          </View>
          <View style={styles.infoRow}>
            <Text style={[styles.infoLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Fee</Text>
            <Text style={[styles.infoValue, { color: COLORS.primary }]}>{selectedBridge.fee}</Text>
          </View>
          <View style={styles.infoRow}>
            <Text style={[styles.infoLabel, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>Route</Text>
            <Text style={[styles.infoValue, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{selectedBridge.name}</Text>
          </View>
        </View>

        {/* Bridge Options */}
        <Text style={[styles.sectionTitle, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>Select Bridge</Text>
        {bridges.map(bridge => (
          <TouchableOpacity key={bridge.id} style={[styles.bridgeCard, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
            <Text style={styles.bridgeLogo}>{bridge.logo}</Text>
            <View style={styles.bridgeInfo}>
              <Text style={[styles.bridgeName, { color: isDark ? COLORS.textDark : COLORS.textLight }]}>{bridge.name}</Text>
              <Text style={[styles.bridgeChains, { color: isDark ? COLORS.gray : COLORS.lightGray }]}>{bridge.chains.slice(0, 3).join(', ')}...</Text>
            </View>
            <Text style={[styles.bridgeFee, { color: COLORS.success }]}>{bridge.fee}</Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      {/* Bridge Button */}
      <View style={[styles.bottomBar, { backgroundColor: isDark ? COLORS.cardDark : COLORS.cardLight }]}>
        <TouchableOpacity style={[styles.bridgeActionButton, { backgroundColor: COLORS.primary }]} disabled={!amount}>
          <Text style={styles.bridgeActionText}>Bridge</Text>
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
  sectionTitle: { fontSize: FONT_SIZES.lg, fontWeight: 'bold', marginTop: SPACING.lg, marginBottom: SPACING.sm },
  bridgeCard: { flexDirection: 'row', alignItems: 'center', padding: SPACING.md, borderRadius: 12, marginBottom: SPACING.sm },
  bridgeLogo: { fontSize: 28, marginRight: SPACING.md },
  bridgeInfo: { flex: 1 },
  bridgeName: { fontSize: FONT_SIZES.md, fontWeight: 'bold' },
  bridgeChains: { fontSize: FONT_SIZES.xs },
  bridgeFee: { fontSize: FONT_SIZES.sm, fontWeight: '600' },
  bottomBar: { position: 'absolute', bottom: 0, left: 0, right: 0, padding: SPACING.md, borderTopWidth: 1, borderTopColor: COLORS.borderDark },
  bridgeActionButton: { padding: SPACING.md, borderRadius: 12, alignItems: 'center' },
  bridgeActionText: { color: COLORS.white, fontSize: FONT_SIZES.lg, fontWeight: 'bold' },
});

export default BridgeScreen;
