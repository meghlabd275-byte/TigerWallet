// ============================================================================
// TigerWallet - Add Wallet Screen
// Create/Import Wallet with Real Crypto
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
import { cryptoService } from '../services/CryptoService';
import { useNavigation } from '@react-navigation/native';

const AddWalletScreen: React.FC = () => {
  const { theme, isDark } = useThemeStore();
  const navigation = useNavigation();
  
  const [mode, setMode] = useState<'create' | 'import'>('create');
  const [walletName, setWalletName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [showMnemonic, setShowMnemonic] = useState(false);

  const generateWallet = async () => {
    if (!walletName.trim()) {
      Alert.alert('Error', 'Please enter a wallet name');
      return;
    }
    if (password.length < 8) {
      Alert.alert('Error', 'Password must be at least 8 characters');
      return;
    }
    if (password !== confirmPassword) {
      Alert.alert('Error', 'Passwords do not match');
      return;
    }

    setLoading(true);
    try {
      const newMnemonic = cryptoService.generateMnemonic(256);
      setMnemonic(newMnemonic);
      setStep(2);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to create wallet');
    } finally {
      setLoading(false);
    }
  };

  const confirmBackup = async () => {
    if (!password || !confirmPassword) {
      Alert.alert('Error', 'Please set a password');
      return;
    }

    setLoading(true);
    try {
      const wallet = await walletService.createWallet(walletName, password);
      await walletService.setActiveWallet(wallet.id);
      Alert.alert('Success', 'Wallet created successfully!', [
        { text: 'OK', onPress: () => navigation.goBack() }
      ]);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to create wallet');
    } finally {
      setLoading(false);
    }
  };

  const importWallet = async () => {
    if (!walletName.trim()) {
      Alert.alert('Error', 'Please enter a wallet name');
      return;
    }
    if (password.length < 8) {
      Alert.alert('Error', 'Password must be at least 8 characters');
      return;
    }

    setLoading(true);
    try {
      let wallet;
      
      if (mnemonic.trim()) {
        // Import from mnemonic
        if (!cryptoService.validateMnemonic(mnemonic)) {
          throw new Error('Invalid mnemonic phrase');
        }
        wallet = await walletService.importWallet(mnemonic.trim(), walletName, password);
      } else if (privateKey.trim()) {
        // Import from private key
        wallet = await walletService.importFromPrivateKey(privateKey.trim(), walletName, password);
      } else {
        throw new Error('Please enter a mnemonic phrase or private key');
      }

      await walletService.setActiveWallet(wallet.id);
      Alert.alert('Success', 'Wallet imported successfully!', [
        { text: 'OK', onPress: () => navigation.goBack() }
      ]);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to import wallet');
    } finally {
      setLoading(false);
    }
  };

  const renderCreateMode = () => (
    <View style={styles.stepContainer}>
      {step === 1 && (
        <>
          <Text style={[styles.stepTitle, { color: theme.colors.text }]}>
            Create New Wallet
          </Text>
          <Text style={[styles.stepDescription, { color: theme.colors.textSecondary }]}>
            Your wallet will be secured with a 24-word recovery phrase
          </Text>

          <View style={styles.inputContainer}>
            <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
              Wallet Name
            </Text>
            <TextInput
              style={[styles.input, { 
                backgroundColor: theme.colors.surfaceVariant,
                color: theme.colors.text,
                borderColor: theme.colors.border,
              }]}
              placeholder="My Wallet"
              placeholderTextColor={theme.colors.textTertiary}
              value={walletName}
              onChangeText={setWalletName}
            />
          </View>

          <View style={styles.inputContainer}>
            <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
              Password
            </Text>
            <TextInput
              style={[styles.input, { 
                backgroundColor: theme.colors.surfaceVariant,
                color: theme.colors.text,
                borderColor: theme.colors.border,
              }]}
              placeholder="Min 8 characters"
              placeholderTextColor={theme.colors.textTertiary}
              value={password}
              onChangeText={setPassword}
              secureTextEntry
            />
          </View>

          <View style={styles.inputContainer}>
            <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
              Confirm Password
            </Text>
            <TextInput
              style={[styles.input, { 
                backgroundColor: theme.colors.surfaceVariant,
                color: theme.colors.text,
                borderColor: theme.colors.border,
              }]}
              placeholder="Confirm password"
              placeholderTextColor={theme.colors.textTertiary}
              value={confirmPassword}
              onChangeText={setConfirmPassword}
              secureTextEntry
            />
          </View>

          <TouchableOpacity
            style={[styles.primaryButton, { backgroundColor: theme.colors.primary }]}
            onPress={generateWallet}
            disabled={loading}
          >
            <Text style={styles.primaryButtonText}>
              {loading ? 'Creating...' : 'Create Wallet'}
            </Text>
          </TouchableOpacity>
        </>
      )}

      {step === 2 && (
        <>
          <Text style={[styles.stepTitle, { color: theme.colors.text }]}>
            Your Recovery Phrase
          </Text>
          <Text style={[styles.warningText, { color: theme.colors.warning }]}>
            ⚠️ Write down these words in order and store them safely
          </Text>

          <View style={[styles.mnemonicContainer, { backgroundColor: theme.colors.surface }]}>
            <ScrollView style={styles.mnemonicScroll}>
              <View style={styles.mnemonicGrid}>
                {mnemonic.split(' ').map((word, index) => (
                  <View 
                    key={index} 
                    style={[styles.mnemonicWord, { backgroundColor: theme.colors.surfaceVariant }]}
                  >
                    <Text style={[styles.mnemonicIndex, { color: theme.colors.textTertiary }]}>
                      {index + 1}.
                    </Text>
                    <Text style={[styles.mnemonicText, { color: theme.colors.text }]}>
                      {word}
                    </Text>
                  </View>
                ))}
              </View>
            </ScrollView>
          </View>

          <View style={styles.checkboxContainer}>
            <TouchableOpacity 
              style={[styles.checkbox, { borderColor: theme.colors.primary }]}
              onPress={() => setShowMnemonic(!showMnemonic)}
            >
              {showMnemonic && <Text style={styles.checkmark}>✓</Text>}
            </TouchableOpacity>
            <Text style={[styles.checkboxLabel, { color: theme.colors.textSecondary }]}>
              I have securely stored my recovery phrase
            </Text>
          </View>

          <TouchableOpacity
            style={[
              styles.primaryButton, 
              { backgroundColor: theme.colors.primary, opacity: showMnemonic ? 1 : 0.5 }
            ]}
            onPress={confirmBackup}
            disabled={!showMnemonic || loading}
          >
            <Text style={styles.primaryButtonText}>
              {loading ? 'Saving...' : 'I Have Saved My Phrase'}
            </Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.secondaryButton, { borderColor: theme.colors.border }]}
            onPress={() => setStep(1)}
          >
            <Text style={[styles.secondaryButtonText, { color: theme.colors.text }]}>
              Go Back
            </Text>
          </TouchableOpacity>
        </>
      )}
    </View>
  );

  const renderImportMode = () => (
    <View style={styles.stepContainer}>
      <Text style={[styles.stepTitle, { color: theme.colors.text }]}>
        Import Wallet
      </Text>

      <View style={styles.importOptions}>
        <TouchableOpacity
          style={[styles.importOption, { backgroundColor: theme.colors.surface }]}
          onPress={() => setMnemonic('')}
        >
          <Text style={styles.importIcon}>📝</Text>
          <Text style={[styles.importTitle, { color: theme.colors.text }]}>
            Mnemonic Phrase
          </Text>
          <Text style={[styles.importSubtitle, { color: theme.colors.textSecondary }]}>
            12 or 24 words
          </Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={[styles.importOption, { backgroundColor: theme.colors.surface }]}
          onPress={() => setMnemonic('')}
        >
          <Text style={styles.importIcon}>🔑</Text>
          <Text style={[styles.importTitle, { color: theme.colors.text }]}>
            Private Key
          </Text>
          <Text style={[styles.importSubtitle, { color: theme.colors.textSecondary }]}>
            Hex format
          </Text>
        </TouchableOpacity>
      </View>

      <View style={styles.inputContainer}>
        <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
          Wallet Name
        </Text>
        <TextInput
          style={[styles.input, { 
            backgroundColor: theme.colors.surfaceVariant,
            color: theme.colors.text,
            borderColor: theme.colors.border,
          }]}
          placeholder="My Wallet"
          placeholderTextColor={theme.colors.textTertiary}
          value={walletName}
          onChangeText={setWalletName}
        />
      </View>

      {mnemonic ? (
        <View style={styles.inputContainer}>
          <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
            Recovery Phrase
          </Text>
          <TextInput
            style={[styles.input, styles.multilineInput, { 
              backgroundColor: theme.colors.surfaceVariant,
              color: theme.colors.text,
              borderColor: theme.colors.border,
            }]}
            placeholder="word1 word2 word3 ..."
            placeholderTextColor={theme.colors.textTertiary}
            value={mnemonic}
            onChangeText={setMnemonic}
            multiline
            numberOfLines={4}
          />
        </View>
      ) : (
        <View style={styles.inputContainer}>
          <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
            Private Key
          </Text>
          <TextInput
            style={[styles.input, { 
              backgroundColor: theme.colors.surfaceVariant,
              color: theme.colors.text,
              borderColor: theme.colors.border,
            }]}
            placeholder="0x..."
            placeholderTextColor={theme.colors.textTertiary}
            value={privateKey}
            onChangeText={setPrivateKey}
            autoCapitalize="none"
          />
        </View>
      )}

      <View style={styles.inputContainer}>
        <Text style={[styles.inputLabel, { color: theme.colors.textSecondary }]}>
          Password
        </Text>
        <TextInput
          style={[styles.input, { 
            backgroundColor: theme.colors.surfaceVariant,
            color: theme.colors.text,
            borderColor: theme.colors.border,
          }]}
          placeholder="Min 8 characters"
          placeholderTextColor={theme.colors.textTertiary}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
        />
      </View>

      <TouchableOpacity
        style={[styles.primaryButton, { backgroundColor: theme.colors.primary }]}
        onPress={importWallet}
        disabled={loading}
      >
        <Text style={styles.primaryButtonText}>
          {loading ? 'Importing...' : 'Import Wallet'}
        </Text>
      </TouchableOpacity>
    </View>
  );

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
          Add Wallet
        </Text>
        <View style={{ width: 50 }} />
      </View>

      <View style={styles.tabContainer}>
        <TouchableOpacity
          style={[
            styles.tab,
            mode === 'create' && { borderBottomColor: theme.colors.primary }
          ]}
          onPress={() => setMode('create')}
        >
          <Text style={[
            styles.tabText,
            { color: mode === 'create' ? theme.colors.primary : theme.colors.textSecondary }
          ]}>
            Create New
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[
            styles.tab,
            mode === 'import' && { borderBottomColor: theme.colors.primary }
          ]}
          onPress={() => setMode('import')}
        >
          <Text style={[
            styles.tabText,
            { color: mode === 'import' ? theme.colors.primary : theme.colors.textSecondary }
          ]}>
            Import
          </Text>
        </TouchableOpacity>
      </View>

      <ScrollView 
        style={styles.content}
        showsVerticalScrollIndicator={false}
        keyboardShouldPersistTaps="handled"
      >
        {mode === 'create' ? renderCreateMode() : renderImportMode()}
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 50,
    paddingHorizontal: 20,
    paddingBottom: 20,
  },
  backButton: {
    fontSize: 16,
    fontWeight: '600',
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: '600',
  },
  tabContainer: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: '#E9ECEF',
  },
  tab: {
    flex: 1,
    paddingVertical: 16,
    alignItems: 'center',
    borderBottomWidth: 2,
    borderBottomColor: 'transparent',
  },
  tabText: {
    fontSize: 16,
    fontWeight: '600',
  },
  content: {
    flex: 1,
  },
  stepContainer: {
    padding: 20,
  },
  stepTitle: {
    fontSize: 24,
    fontWeight: '700',
    marginBottom: 8,
    textAlign: 'center',
  },
  stepDescription: {
    fontSize: 14,
    textAlign: 'center',
    marginBottom: 24,
  },
  warningText: {
    fontSize: 14,
    textAlign: 'center',
    marginBottom: 20,
    fontWeight: '600',
  },
  inputContainer: {
    marginBottom: 16,
  },
  inputLabel: {
    fontSize: 12,
    fontWeight: '500',
    marginBottom: 8,
  },
  input: {
    padding: 16,
    borderRadius: 12,
    fontSize: 16,
    borderWidth: 1,
  },
  multilineInput: {
    minHeight: 100,
    textAlignVertical: 'top',
  },
  primaryButton: {
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  primaryButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
  secondaryButton: {
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    marginTop: 12,
    borderWidth: 1,
  },
  secondaryButtonText: {
    fontSize: 16,
    fontWeight: '600',
  },
  mnemonicContainer: {
    borderRadius: 12,
    padding: 16,
    marginBottom: 20,
  },
  mnemonicScroll: {
    maxHeight: 200,
  },
  mnemonicGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
  },
  mnemonicWord: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 8,
    marginBottom: 8,
    width: '48%',
  },
  mnemonicIndex: {
    fontSize: 10,
    marginRight: 6,
  },
  mnemonicText: {
    fontSize: 12,
    fontWeight: '600',
  },
  checkboxContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 20,
  },
  checkbox: {
    width: 24,
    height: 24,
    borderRadius: 6,
    borderWidth: 2,
    marginRight: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  checkmark: {
    fontSize: 14,
    fontWeight: '700',
  },
  checkboxLabel: {
    fontSize: 14,
    flex: 1,
  },
  importOptions: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 24,
  },
  importOption: {
    flex: 1,
    padding: 20,
    borderRadius: 12,
    alignItems: 'center',
  },
  importIcon: {
    fontSize: 32,
    marginBottom: 8,
  },
  importTitle: {
    fontSize: 14,
    fontWeight: '600',
  },
  importSubtitle: {
    fontSize: 12,
    marginTop: 4,
  },
});

export default AddWalletScreen;
