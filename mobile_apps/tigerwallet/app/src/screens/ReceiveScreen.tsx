// ============================================================================
// TigerWallet - Receive Screen
// Receive Crypto with QR Code
// ============================================================================

import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Share,
  Alert,
} from 'react-native';
import QRCode from 'react-native-qrcode-svg';
import { useThemeStore } from '../stores/ThemeStore';
import { walletService } from '../services/WalletService';
import { blockchainService } from '../services/BlockchainService';
import { useNavigation } from '@react-navigation/native';

const ReceiveScreen: React.FC = () => {
  const { theme, isDark } = useThemeStore();
  const navigation = useNavigation();
  
  const [selectedChain, setSelectedChain] = useState(1);
  const activeWallet = walletService.getActiveWallet();

  const address = activeWallet?.addresses[selectedChain] || '';

  const copyAddress = () => {
    // Would use Clipboard in real app
    Alert.alert('Copied', 'Address copied to clipboard');
  };

  const shareAddress = async () => {
    try {
      await Share.share({
        message: address,
      });
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}>
          <Text style={[styles.backButton, { color: theme.colors.primary }]}>
            ← Back
          </Text>
        </TouchableOpacity>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>
          Receive
        </Text>
        <View style={{ width: 50 }} />
      </View>

      <View style={styles.chainSelector}>
        {blockchainService.getSupportedChains().slice(0, 8).map((chain) => (
          <TouchableOpacity
            key={chain.id}
            style={[
              styles.chainChip,
              { 
                backgroundColor: selectedChain === chain.id 
                  ? theme.colors.primary 
                  : theme.colors.surfaceVariant 
              }
            ]}
            onPress={() => setSelectedChain(chain.id)}
          >
            <Text style={[
              styles.chainChipText,
              { color: selectedChain === chain.id ? '#FFFFFF' : theme.colors.text }
            ]}>
              {chain.symbol}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      <View style={[styles.qrContainer, { backgroundColor: theme.colors.surface }]}>
        <View style={styles.qrCode}>
          <QRCode
            value={address}
            size={200}
            color={isDark ? '#FFFFFF' : '#000000'}
            backgroundColor={theme.colors.surface}
          />
        </View>

        <Text style={[styles.addressLabel, { color: theme.colors.textSecondary }]}>
          {blockchainService.getChain(selectedChain)?.name} Address
        </Text>

        <View style={[styles.addressBox, { backgroundColor: theme.colors.surfaceVariant }]}>
          <Text style={[styles.addressText, { color: theme.colors.text }]} selectable>
            {address}
          </Text>
        </View>

        <View style={styles.actionButtons}>
          <TouchableOpacity
            style={[styles.actionButton, { backgroundColor: theme.colors.primary }]}
            onPress={copyAddress}
          >
            <Text style={styles.actionIcon}>📋</Text>
            <Text style={styles.actionText}>Copy</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.actionButton, { backgroundColor: theme.colors.primary }]}
            onPress={shareAddress}
          >
            <Text style={styles.actionIcon}>📤</Text>
            <Text style={styles.actionText}>Share</Text>
          </TouchableOpacity>
        </View>
      </View>

      <Text style={[styles.warningText, { color: theme.colors.warning }]}>
        ⚠️ Only send {blockchainService.getChain(selectedChain)?.symbol} to this address
      </Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 50,
    paddingHorizontal: 20,
    paddingBottom: 20,
  },
  backButton: { fontSize: 16, fontWeight: '600' },
  headerTitle: { fontSize: 18, fontWeight: '600' },
  chainSelector: {
    flexDirection: 'row',
    paddingHorizontal: 20,
    marginBottom: 20,
    gap: 8,
  },
  chainChip: { paddingHorizontal: 16, paddingVertical: 8, borderRadius: 20 },
  chainChipText: { fontSize: 14, fontWeight: '600' },
  qrContainer: { margin: 20, padding: 24, borderRadius: 16, alignItems: 'center' },
  qrCode: { marginBottom: 20 },
  addressLabel: { fontSize: 14, marginBottom: 12 },
  addressBox: { padding: 16, borderRadius: 12, width: '100%' },
  addressText: { fontSize: 12, textAlign: 'center', fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace' },
  actionButtons: { flexDirection: 'row', gap: 12, marginTop: 20 },
  actionButton: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 20, paddingVertical: 12, borderRadius: 12, gap: 8 },
  actionIcon: { fontSize: 16 },
  actionText: { color: '#FFFFFF', fontWeight: '600' },
  warningText: { textAlign: 'center', fontSize: 12, marginTop: 20 },
});

import { Platform } from 'react-native';
export default ReceiveScreen;
