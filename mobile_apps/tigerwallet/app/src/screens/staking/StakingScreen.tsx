import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, Alert, ActivityIndicator } from 'react-native';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { API } from '../../services/API';

interface StakingPool {
  id: string;
  name: string;
  apy: string;
  staked: string;
  reward: string;
  token: string;
}

// Staking Screen — fetches real staking pools/positions from the canonical
// wallet_api staking endpoints (no hardcoded mock pools).
const StakingScreen = () => {
  const wallet = useSelector((state: RootState) => state.wallet.wallet);
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';

  const [selectedTab, setSelectedTab] = useState('Stake');
  const [stakeAmount, setStakeAmount] = useState('');
  const [selectedPool, setSelectedPool] = useState<string>('');
  const [isStaking, setIsStaking] = useState(false);
  const [pools, setPools] = useState<StakingPool[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadPools = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await API.getStakingPools(1);
      const list = res?.data?.pools ?? res?.data ?? [];
      const mapped: StakingPool[] = (list as any[]).map((p) => ({
        id: p.id ?? p.pool_id ?? p.asset ?? p.name,
        name: p.name ?? p.asset ?? p.token ?? 'Pool',
        apy: `${((p.apy ?? p.apr ?? 0) as number).toFixed(2)}%`,
        staked: p.staked ?? p.staked_amount ?? `0 ${p.asset ?? ''}`,
        reward: p.reward ?? p.pending_reward ?? `0 ${p.asset ?? ''}`,
        token: p.asset ?? p.token ?? '',
      }));
      setPools(mapped);
      if (mapped.length > 0 && !selectedPool) setSelectedPool(mapped[0].name);
    } catch (err) {
      setError('Failed to load staking pools. Pull down to retry.');
      setPools([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPools();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleStake = async () => {
    if (!stakeAmount || parseFloat(stakeAmount) <= 0) {
      Alert.alert('Error', 'Please enter valid amount');
      return;
    }
    if (!wallet?.id || !selectedPool) {
      Alert.alert('Error', 'No wallet or pool selected');
      return;
    }
    const pool = pools.find((p) => p.name === selectedPool);
    if (!pool) {
      Alert.alert('Error', 'Invalid pool');
      return;
    }
    setIsStaking(true);
    try {
      const res = await API.stake({
        walletId: wallet.id,
        poolId: pool.id,
        amount: stakeAmount,
      });
      if (res?.success === false) {
        Alert.alert('Error', res.error || 'Staking failed');
      } else {
        Alert.alert('Success', 'Staked successfully!');
        setStakeAmount('');
        await loadPools();
      }
    } catch (err) {
      Alert.alert('Error', 'Staking request failed');
    } finally {
      setIsStaking(false);
    }
  };

  const handleClaim = async (pool: StakingPool) => {
    if (!wallet?.id) {
      Alert.alert('Error', 'No wallet connected');
      return;
    }
    try {
      const res = await API.unstake({ walletId: wallet.id, positionId: pool.id });
      if (res?.success === false) {
        Alert.alert('Error', res.error || 'Claim failed');
      } else {
        Alert.alert('Success', `Claimed rewards from ${pool.name}`);
        await loadPools();
      }
    } catch (err) {
      Alert.alert('Error', 'Claim request failed');
    }
  };

  return (
    <ScrollView style={[styles.container, { backgroundColor: isDark ? '#0f172a' : '#fff' }]}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={[styles.title, { color: isDark ? '#fff' : '#000' }]}>Staking</Text>
      </View>

      {/* Loading / error state */}
      {loading && (
        <View style={styles.stateContainer}>
          <ActivityIndicator color="#f97316" />
          <Text style={{ color: isDark ? '#94a3b8' : '#666', marginTop: 8 }}>Loading pools...</Text>
        </View>
      )}
      {error && !loading && (
        <View style={styles.stateContainer}>
          <Text style={{ color: '#ef4444', textAlign: 'center' }}>{error}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={loadPools}>
            <Text style={styles.retryButtonText}>Retry</Text>
          </TouchableOpacity>
        </View>
      )}

      {!loading && !error && pools.length === 0 && (
        <View style={styles.stateContainer}>
          <Text style={{ color: isDark ? '#94a3b8' : '#666', textAlign: 'center' }}>No staking pools available.</Text>
        </View>
      )}

      {/* Tab Selector */}
      <View style={[styles.tabs, { borderBottomColor: isDark ? '#1e293b' : '#eee' }]}>
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

      {selectedTab === 'Stake' && pools.length > 0 && (
        <View style={styles.content}>
          {/* Pool Selector */}
          <View style={[styles.card, { backgroundColor: isDark ? '#1e293b' : '#f9f9f9' }]}>
            <Text style={[styles.label, { color: isDark ? '#94a3b8' : '#666' }]}>Select Pool</Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false}>
              <View style={styles.poolList}>
                {pools.map((pool) => (
                  <TouchableOpacity
                    key={pool.id}
                    style={[styles.poolChip, selectedPool === pool.name && styles.poolChipActive, { borderColor: isDark ? '#334155' : '#eee' }]}
                    onPress={() => setSelectedPool(pool.name)}
                  >
                    <Text style={[styles.poolName, { color: isDark ? '#fff' : '#000' }]}>{pool.name}</Text>
                    <Text style={styles.poolAPY}>APY: {pool.apy}</Text>
                  </TouchableOpacity>
                ))}
              </View>
            </ScrollView>
          </View>

          {/* Amount */}
          <View style={[styles.card, { backgroundColor: isDark ? '#1e293b' : '#f9f9f9' }]}>
            <View style={styles.amountHeader}>
              <Text style={[styles.label, { color: isDark ? '#94a3b8' : '#666' }]}>Amount</Text>
              <TouchableOpacity onPress={() => setStakeAmount('1.0')}>
                <Text style={styles.maxBtn}>MAX</Text>
              </TouchableOpacity>
            </View>
            <View style={styles.inputContainer}>
              <TextInput
                style={[styles.input, { color: isDark ? '#fff' : '#000' }]}
                value={stakeAmount}
                onChangeText={setStakeAmount}
                placeholder="0.0"
                keyboardType="decimal-pad"
                placeholderTextColor={isDark ? '#475569' : '#999'}
              />
              <Text style={[styles.tokenLabel, { color: isDark ? '#94a3b8' : '#666' }]}>{selectedPool.split(' ')[0]}</Text>
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
            <View key={pool.id} style={[styles.earnCard, { backgroundColor: isDark ? '#1e293b' : '#f9f9f9' }]}>
              <View style={styles.earnHeader}>
                <View>
                  <Text style={[styles.poolName, { color: isDark ? '#fff' : '#000' }]}>{pool.name}</Text>
                  <Text style={styles.apyGreen}>APY: {pool.apy}</Text>
                </View>
                <View style={styles.earnRight}>
                  <Text style={[styles.stakedLabel, { color: isDark ? '#94a3b8' : '#666' }]}>Staked</Text>
                  <Text style={[styles.stakedValue, { color: isDark ? '#fff' : '#000' }]}>{pool.staked}</Text>
                </View>
              </View>
              <View style={[styles.earnFooter, { borderTopColor: isDark ? '#334155' : '#eee' }]}>
                <View>
                  <Text style={[styles.rewardLabel, { color: isDark ? '#94a3b8' : '#666' }]}>Pending Reward</Text>
                  <Text style={styles.rewardValue}>{pool.reward}</Text>
                </View>
                <TouchableOpacity
                  style={styles.claimButton}
                  onPress={() => handleClaim(pool)}
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
            <View key={pool.id} style={[styles.poolCard, { backgroundColor: isDark ? '#1e293b' : '#f9f9f9' }]}>
              <View style={styles.poolCardHeader}>
                <Text style={[styles.poolCardName, { color: isDark ? '#fff' : '#000' }]}>{pool.name}</Text>
                <View style={styles.apyContainer}>
                  <Text style={styles.apyValue}>{pool.apy}</Text>
                  <Text style={[styles.apyLabel, { color: isDark ? '#94a3b8' : '#666' }]}>APY</Text>
                </View>
              </View>
              <Text style={[styles.totalStaked, { color: isDark ? '#94a3b8' : '#666' }]}>Total Staked: {pool.staked}</Text>
              <TouchableOpacity style={styles.stakePoolButton} onPress={() => { setSelectedTab('Stake'); setSelectedPool(pool.name); }}>
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
  stateContainer: {
    padding: 24,
    alignItems: 'center',
  },
  retryButton: {
    marginTop: 12,
    backgroundColor: '#f97316',
    paddingHorizontal: 24,
    paddingVertical: 10,
    borderRadius: 8,
  },
  retryButtonText: {
    color: '#fff',
    fontWeight: '600',
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
