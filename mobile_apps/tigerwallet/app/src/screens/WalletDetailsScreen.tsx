import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView } from 'react-native';
import { useThemeStore } from '../stores/ThemeStore';
import { useNavigation } from '@react-navigation/native';
const WalletDetailsScreen: React.FC = () => {
  const { theme } = useThemeStore();
  const navigation = useNavigation();
  return (
    <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}><Text style={[styles.backButton, { color: theme.colors.primary }]}>← Back</Text></TouchableOpacity>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>Wallet Details</Text>
        <View style={{ width: 50 }} />
      </View>
      <ScrollView style={styles.content}>
        <View style={[styles.card, { backgroundColor: theme.colors.surface }]}>
          <Text style={[styles.label, { color: theme.colors.textSecondary }]}>Address</Text>
          <Text style={[styles.address, { color: theme.colors.text }]}>0x742d35Cc6634C0532925a3b844Bc9e7595f...</Text>
        </View>
      </ScrollView>
    </View>
  );
};
const styles = StyleSheet.create({
  container: { flex: 1 }, header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingTop: 50, paddingHorizontal: 20, paddingBottom: 20 }, backButton: { fontSize: 16, fontWeight: '600' }, headerTitle: { fontSize: 18, fontWeight: '600' }, content: { flex: 1, padding: 20 }, card: { padding: 16, borderRadius: 12, marginBottom: 16 }, label: { fontSize: 12, marginBottom: 8 }, address: { fontSize: 14, fontFamily: 'monospace' },
});
export default WalletDetailsScreen;
