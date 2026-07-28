// Swap Screen
import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView } from 'react-native';
import { useThemeStore } from '../stores/ThemeStore';
import { useNavigation } from '@react-navigation/native';

const SwapScreen: React.FC = () => {
  const { theme } = useThemeStore();
  const navigation = useNavigation();

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}>
          <Text style={[styles.backButton, { color: theme.colors.primary }]}>← Back</Text>
        </TouchableOpacity>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>Swap</Text>
        <View style={{ width: 50 }} />
      </View>
      <ScrollView style={styles.content}>
        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>From</Text>
          <View style={styles.tokenRow}>
            <Text style={[styles.tokenSymbol, { color: theme.colors.text }]}>ETH</Text>
            <Text style={[styles.balance, { color: theme.colors.textTertiary }]}>0.00</Text>
          </View>
          <View style={[styles.input, { backgroundColor: theme.colors.surfaceVariant }]}>
            <Text style={[styles.inputText, { color: theme.colors.text }]}>0.00</Text>
          </View>
        </View>
        
        <TouchableOpacity style={[styles.swapButton, { backgroundColor: theme.colors.primary }]}>
          <Text style={styles.swapIcon}>⇅</Text>
        </TouchableOpacity>

        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>To</Text>
          <View style={styles.tokenRow}>
            <Text style={[styles.tokenSymbol, { color: theme.colors.text }]}>USDT</Text>
          </View>
          <View style={[styles.input, { backgroundColor: theme.colors.surfaceVariant }]}>
            <Text style={[styles.inputText, { color: theme.colors.text }]}>0.00</Text>
          </View>
        </View>

        <View style={[styles.rateInfo, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.rateLabel, { color: theme.colors.textSecondary }]}>Rate</Text>
          <Text style={[styles.rateValue, { color: theme.colors.text }]}>1 ETH = 2,500 USDT</Text>
        </View>

        <TouchableOpacity style={[styles.swapButtonFull, { backgroundColor: theme.colors.primary }]}>
          <Text style={styles.swapButtonText}>Swap</Text>
        </TouchableOpacity>
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
  label: { fontSize: 12, fontWeight: '500', marginBottom: 12 },
  tokenRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 12 },
  tokenSymbol: { fontSize: 18, fontWeight: '600' },
  balance: { fontSize: 14 },
  input: { padding: 16, borderRadius: 12 },
  inputText: { fontSize: 24, fontWeight: '700' },
  swapButton: { width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center', alignSelf: 'center', marginVertical: -12, zIndex: 10 },
  swapIcon: { fontSize: 20, color: '#FFF' },
  rateInfo: { padding: 16, borderRadius: 12, marginBottom: 24 },
  rateLabel: { fontSize: 12, marginBottom: 4 },
  rateValue: { fontSize: 14, fontWeight: '500' },
  swapButtonFull: { padding: 18, borderRadius: 12, alignItems: 'center' },
  swapButtonText: { color: '#FFF', fontSize: 18, fontWeight: '700' },
});

export default SwapScreen;
