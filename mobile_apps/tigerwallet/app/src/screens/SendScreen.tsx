// ============================================================================
// TigerWallet - Send Screen
// Send Transactions with Real Blockchain
// ============================================================================

import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  ScrollView,
  Alert,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { useThemeStore } from '../stores/ThemeStore';
import { walletService } from '../services/WalletService';
import { blockchainService } from '../services/BlockchainService';
import { cryptoService } from '../services/CryptoService';
import { useNavigation } from '@react-navigation/native';

const SendScreen: React.FC = () => {
  const { theme, isDark } = useThemeStore();
  const navigation = useNavigation();
  
  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedChain, setSelectedChain] = useState(1);
  const [selectedToken, setSelectedToken] = useState('native');
  const [loading, setLoading] = useState(false);

  const activeWallet = walletService.getActiveWallet();

  const sendTransaction = async () => {
    if (!activeWallet) {
      Alert.alert('Error', 'No wallet selected');
      return;
    }

    if (!recipient) {
      Alert.alert('Error', 'Please enter a recipient address');
      return;
    }

    if (!blockchainService.isValidAddress(recipient)) {
      Alert.alert('Error', 'Invalid recipient address');
      return;
    }

    if (!amount || parseFloat(amount) <= 0) {
      Alert.alert('Error', 'Please enter a valid amount');
      return;
    }

    setLoading(true);
    try {
      const tx = await walletService.sendTransaction(
        activeWallet.id,
        selectedChain,
        recipient,
        amount
      );
      
      Alert.alert(
        'Transaction Sent',
        `Transaction hash: ${blockchainService.formatAddress(tx.hash)}`,
        [{ text: 'OK', onPress: () => navigation.goBack() }]
      );
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to send transaction');
    } finally {
      setLoading(false);
    }
  };

  const maxAmount = async () => {
    if (!activeWallet) return;
    try {
      const account = await walletService.getAccount(activeWallet.id, selectedChain);
      const balance = cryptoService.formatEther(account.balance);
      setAmount(balance);
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <KeyboardAvoidingView 
      style={[styles.container, { backgroundColor: theme.colors.background }]}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}>
          <Text style={[styles.backButton, { color: theme.colors.primary }]}>
            ← Back
          </Text>
        </TouchableOpacity>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>
          Send
        </Text>
        <View style={{ width: 50 }} />
      </View>

      <ScrollView style={styles.content} keyboardShouldPersistTaps="handled">
        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>
            From Chain
          </Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
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
          </ScrollView>
        </View>

        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>
            Recipient Address
          </Text>
          <View style={{ flexDirection: 'row', alignItems: 'center' }}>
            <TextInput
              style={[styles.input, { 
                backgroundColor: theme.colors.surfaceVariant,
                color: theme.colors.text,
                borderColor: theme.colors.border,
                flex: 1,
              }]}
              placeholder="0x... or tap QR to scan"
              placeholderTextColor={theme.colors.textTertiary}
              value={recipient}
              onChangeText={setRecipient}
              autoCapitalize="none"
            />
            <TouchableOpacity 
              style={[styles.qrButton, { backgroundColor: theme.colors.primary }]}
              onPress={() => {
                // For demo - in production would open camera QR scanner
                Alert.alert(
                  'QR Scanner',
                  'Camera QR scanning would open here.\n\nSupported chains:\n• Ethereum\n• Bitcoin\n• Solana\n• TRON\n• And more...',
                  [
                    { text: 'Demo: Paste Sample Address', onPress: () => setRecipient('0x742d35Cc6634C0532925a3b844Bc9e7595f1234') },
                    { text: 'Cancel', style: 'cancel' }
                  ]
                );
              }}
            >
              <Text style={styles.qrButtonText}>📷</Text>
            </TouchableOpacity>
          </View>
        </View>

        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <View style={styles.amountHeader}>
            <Text style={[styles.label, { color: theme.colors.textSecondary }]}>
              Amount
            </Text>
            <TouchableOpacity onPress={maxAmount}>
              <Text style={[styles.maxButton, { color: theme.colors.primary }]}>
                MAX
              </Text>
            </TouchableOpacity>
          </View>
          <TextInput
            style={[styles.input, styles.amountInput, { 
              backgroundColor: theme.colors.surfaceVariant,
              color: theme.colors.text,
              borderColor: theme.colors.border,
            }]}
            placeholder="0.00"
            placeholderTextColor={theme.colors.textTertiary}
            value={amount}
            onChangeText={setAmount}
            keyboardType="decimal-pad"
          />
        </View>

        {activeWallet && (
          <View style={[styles.infoCard, { backgroundColor: theme.colors.surface }]}>
            <Text style={[styles.infoLabel, { color: theme.colors.textSecondary }]}>
              From Address
            </Text>
            <Text style={[styles.infoValue, { color: theme.colors.text }]}>
              {blockchainService.formatAddress(activeWallet.addresses[selectedChain] || '')}
            </Text>
          </View>
        )}

        <TouchableOpacity
          style={[styles.sendButton, { backgroundColor: theme.colors.primary }]}
          onPress={sendTransaction}
          disabled={loading}
        >
          <Text style={styles.sendButtonText}>
            {loading ? 'Sending...' : 'Send'}
          </Text>
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
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
  content: { flex: 1, padding: 20 },
  card: { padding: 16, borderRadius: 12, marginBottom: 16 },
  label: { fontSize: 12, fontWeight: '500', marginBottom: 12 },
  chainChip: { paddingHorizontal: 16, paddingVertical: 8, borderRadius: 20, marginRight: 8 },
  chainChipText: { fontSize: 14, fontWeight: '600' },
  input: { padding: 16, borderRadius: 12, fontSize: 16, borderWidth: 1 },
  amountInput: { fontSize: 24, fontWeight: '700' },
  amountHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 },
  maxButton: { fontSize: 14, fontWeight: '600' },
  infoCard: { padding: 16, borderRadius: 12, marginBottom: 24 },
  infoLabel: { fontSize: 12, marginBottom: 4 },
  infoValue: { fontSize: 14, fontWeight: '500' },
  sendButton: { padding: 18, borderRadius: 12, alignItems: 'center' },
  sendButtonText: { color: '#FFFFFF', fontSize: 18, fontWeight: '700' },
  qrButton: { marginLeft: 12, padding: 14, borderRadius: 12, width: 50, height: 50, justifyContent: 'center', alignItems: 'center' },
  qrButtonText: { fontSize: 20 },
});

export default SendScreen;
