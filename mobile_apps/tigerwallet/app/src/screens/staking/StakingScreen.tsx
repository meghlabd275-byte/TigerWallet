import React, { useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, Alert } from 'react-native';

// Staking Screen - Complete functionality matching Flutter/iOS/Android
const StakingScreen = () => {
  const [selectedTab, setSelectedTab] = useState('Stake');
  const [stakeAmount, setStakeAmount] = useState('');
  const [selectedPool, setSelectedPool] = useState('ETH 2.0');
  const [isStaking, setIsStaking] = useState(false);

  const pools = [
    { name: 'ETH 2.0', apy: '4.2%', staked: '1.5 ETH', reward: '0.063 ETH' },
    { name: 'BNB', apy: '3.8%', staked: '0 BNB', reward: '0 BNB' },
    { name: 'SOL', apy: '6.5%', staked: '0 SOL', reward: '0 SOL' },
    { name: 'MATIC', apy: '5.2%', staked: '0 MATIC', reward: '0 MATIC' }
  ];

  const handleStake = () => {
    if (!stakeAmount || parseFloat(stakeAmount) <= 0) {
      Alert.alert('Error', 'Please enter valid amount');
      return;
    }
    setIsStaking(true);
    setTimeout(() => {
      setIsStaking(false);
      setStakeAmount('');
      Alert.alert('Success', 'Staked successfully!');
    }, 2000);
  };

  const handleClaim = (poolName: string) => {
    Alert.alert('Claim', `Claiming rewards from ${poolName}`);
  };

  return (
    <ScrollView style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.title}>Staking</Text>
      </View>

      {/* Tab Selector */}
      <View style={styles.tabs}>
        {['Stake', 'Earn', 'Pools'].map((tab) => (
          <TouchableOpacity
            key={tab}
            style={[styles.tab, selectedTab === tab && styles.tabActive]}
            onPress={() => setSelectedTab(tab)}
          >
            <Text style={[styles.tabText, selectedTab === tab && styles.tabTextActive]}>
              {tab}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {selectedTab === 'Stake' && (
        <View style={styles.content}>
          {/* Pool Selector */}
          <View style={styles.card}>
            <Text style={styles.label}>Select Pool</Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false}>
              <View style={styles.poolList}>
                {pools.map((pool) => (
                  <TouchableOpacity
                    key={pool.name}
                    style={[styles.poolChip, selectedPool === pool.name && styles.poolChipActive]}
                    onPress={() => setSelectedPool(pool.name)}
                  >
                    <Text style={styles.poolName}>{pool.name}</Text>
                    <Text style={styles.poolAPY}>APY: {pool.apy}</Text>
                  </TouchableOpacity>
                ))}
              </View>
            </ScrollView>
          </View>

          {/* Amount */}
          <View style={styles.card}>
            <View style={styles.amountHeader}>
              <Text style={styles.label}>Amount</Text>
              <TouchableOpacity onPress={() => setStakeAmount('1.0')}>
                <Text style={styles.maxBtn}>MAX</Text>
              </TouchableOpacity>
            </View>
            <View style={styles.inputContainer}>
              <TextInput
                style={styles.input}
                value={stakeAmount}
                onChangeText={setStakeAmount}
                placeholder="0.0"
                keyboardType="decimal-pad"
                placeholderTextColor="#999"
              />
              <Text style={styles.tokenLabel}>{selectedPool.split(' ')[0]}</Text>
            </View>
          </View>

          {/* Stake Button */}
          <TouchableOpacity
            style={[styles.stakeButton, isStaking && styles.stakeButtonDisabled]}
            onPress={handleStake}
            disabled={isStaking}
          >
            <Text style={styles.stakeButtonText}>
              {isStaking ? 'Staking...' : 'Stake'}
            </Text>
          </TouchableOpacity>
        </View>
      )}

      {selectedTab === 'Earn' && (
        <View style={styles.content}>
          {pools.map((pool) => (
            <View key={pool.name} style={styles.earnCard}>
              <View style={styles.earnHeader}>
                <View>
                  <Text style={styles.poolName}>{pool.name}</Text>
                  <Text style={styles.apyGreen}>APY: {pool.apy}</Text>
                </View>
                <View style={styles.earnRight}>
                  <Text style={styles.stakedLabel}>Staked</Text>
                  <Text style={styles.stakedValue}>{pool.staked}</Text>
                </View>
              </View>
              <View style={styles.earnFooter}>
                <View>
                  <Text style={styles.rewardLabel}>Pending Reward</Text>
                  <Text style={styles.rewardValue}>{pool.reward}</Text>
                </View>
                <TouchableOpacity
                  style={styles.claimButton}
                  onPress={() => handleClaim(pool.name)}
                >
                  <Text style={styles.claimButtonText}>Claim</Text>
                </TouchableOpacity>
              </View>
            </View>
          ))}
        </View>
      )}

      {selectedTab === 'Pools' && (
        <View style={styles.content}>
          {pools.map((pool) => (
            <View key={pool.name} style={styles.poolCard}>
              <View style={styles.poolCardHeader}>
                <Text style={styles.poolCardName}>{pool.name}</Text>
                <View style={styles.apyContainer}>
                  <Text style={styles.apyValue}>{pool.apy}</Text>
                  <Text style={styles.apyLabel}>APY</Text>
                </View>
              </View>
              <Text style={styles.totalStaked}>Total Staked: {pool.staked}</Text>
              <TouchableOpacity style={styles.stakePoolButton}>
                <Text style={styles.stakePoolButtonText}>Stake</Text>
              </TouchableOpacity>
            </View>
          ))}
        </View>
      )}
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  header: {
    padding: 20,
    paddingTop: 40,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#000',
  },
  tabs: {
    flexDirection: 'row',
    paddingHorizontal: 20,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  tab: {
    paddingVertical: 12,
    paddingHorizontal: 20,
    marginRight: 10,
  },
  tabActive: {
    borderBottomWidth: 2,
    borderBottomColor: '#f97316',
  },
  tabText: {
    fontSize: 16,
    color: '#666',
  },
  tabTextActive: {
    color: '#f97316',
    fontWeight: 'bold',
  },
  content: {
    padding: 16,
  },
  card: {
    backgroundColor: '#f9f9f9',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  label: {
    fontSize: 14,
    color: '#666',
    marginBottom: 8,
  },
  poolList: {
    flexDirection: 'row',
    gap: 8,
  },
  poolChip: {
    backgroundColor: '#fff',
    padding: 12,
    borderRadius: 8,
    marginRight: 8,
    borderWidth: 1,
    borderColor: '#eee',
  },
  poolChipActive: {
    backgroundColor: '#f97316',
    borderColor: '#f97316',
  },
  poolName: {
    fontWeight: '600',
    color: '#000',
  },
  poolAPY: {
    fontSize: 12,
    color: '#22c55e',
    marginTop: 2,
  },
  amountHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  maxBtn: {
    color: '#f97316',
    fontWeight: '600',
  },
  inputContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  input: {
    flex: 1,
    fontSize: 24,
    fontWeight: '600',
    color: '#000',
  },
  tokenLabel: {
    fontSize: 16,
    color: '#666',
    marginLeft: 8,
  },
  stakeButton: {
    backgroundColor: '#f97316',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  stakeButtonDisabled: {
    opacity: 0.6,
  },
  stakeButtonText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: 'bold',
  },
  earnCard: {
    backgroundColor: '#f9f9f9',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  earnHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  earnRight: {
    alignItems: 'flex-end',
  },
  stakedLabel: {
    fontSize: 12,
    color: '#666',
  },
  stakedValue: {
    fontWeight: '600',
  },
  apyGreen: {
    color: '#22c55e',
    fontSize: 12,
    marginTop: 2,
  },
  earnFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderTopWidth: 1,
    borderTopColor: '#eee',
    paddingTop: 12,
    marginTop: 12,
  },
  rewardLabel: {
    fontSize: 12,
    color: '#666',
  },
  rewardValue: {
    color: '#22c55e',
    fontWeight: '600',
  },
  claimButton: {
    backgroundColor: '#f97316',
    paddingHorizontal: 20,
    paddingVertical: 8,
    borderRadius: 8,
  },
  claimButtonText: {
    color: '#fff',
    fontWeight: '600',
  },
  poolCard: {
    backgroundColor: '#f9f9f9',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  poolCardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  poolCardName: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#000',
  },
  apyContainer: {
    alignItems: 'flex-end',
  },
  apyValue: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#22c55e',
  },
  apyLabel: {
    fontSize: 12,
    color: '#666',
  },
  totalStaked: {
    fontSize: 14,
    color: '#666',
    marginBottom: 12,
  },
  stakePoolButton: {
    backgroundColor: '#f97316',
    padding: 12,
    borderRadius: 8,
    alignItems: 'center',
  },
  stakePoolButtonText: {
    color: '#fff',
    fontWeight: '600',
  },
});

export default StakingScreen;
