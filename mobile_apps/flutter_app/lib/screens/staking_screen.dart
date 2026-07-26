import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Staking Screen - Stake tokens and earn rewards
class StakingScreen extends StatefulWidget {
  const StakingScreen({super.key});

  @override
  State<StakingScreen> createState() => _StakingScreenState();
}

class _StakingScreenState extends State<StakingScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;
  List<Map<String, dynamic>> _stakingPools = [];
  List<Map<String, dynamic>> _userStakes = [];
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _isLoading = true);
    try {
      await Future.wait([
        _loadStakingPools(),
        _loadUserStakes(),
      ]);
    } catch (e) {
      // Handle error
    }
    setState(() => _isLoading = false);
  }

  Future<void> _loadStakingPools() async {
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/staking/pools'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _stakingPools = List<Map<String, dynamic>>.from(data['pools'] ?? []);
        });
      }
    } catch (e) {
      _loadDemoStakingPools();
    }
  }

  void _loadDemoStakingPools() {
    setState(() {
      _stakingPools = [
        {
          'chain': 'Ethereum',
          'token': 'ETH',
          'apr': 4.2,
          'lock_period': 0,
          'min_stake': '0.01',
          'total_staked': '125000',
          'validators': 450000,
          'type': 'liquid'
        },
        {
          'chain': 'Solana',
          'token': 'SOL',
          'apr': 6.8,
          'lock_period': 0,
          'min_stake': '0.01',
          'total_staked': '85000000',
          'validators': 2500,
          'type': 'liquid'
        },
        {
          'chain': 'Polygon',
          'token': 'MATIC',
          'apr': 5.5,
          'lock_period': 0,
          'min_stake': '10',
          'total_staked': '2500000',
          'validators': 3500,
          'type': 'liquid'
        },
        {
          'chain': 'Cosmos',
          'token': 'ATOM',
          'apr': 12.5,
          'lock_period': 21,
          'min_stake': '0.1',
          'total_staked': '150000000',
          'validators': 180,
          'type': 'lock'
        },
        {
          'chain': 'Near',
          'token': 'NEAR',
          'apr': 10.2,
          'lock_period': 0,
          'min_stake': '1',
          'total_staked': '45000000',
          'validators': 120,
          'type': 'liquid'
        },
        {
          'chain': 'Aptos',
          'token': 'APT',
          'apr': 8.5,
          'lock_period': 0,
          'min_stake': '1',
          'total_staked': '150000000',
          'validators': 100,
          'type': 'liquid'
        },
      ];
    });
  }

  Future<void> _loadUserStakes() async {
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/staking/positions'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _userStakes = List<Map<String, dynamic>>.from(data['positions'] ?? []);
        });
      }
    } catch (e) {
      setState(() => _userStakes = []);
    }
  }

  Future<void> _stake(String token, String amount) async {
    try {
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/staking/stake'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'token': token,
          'amount': amount,
        }),
      );

      if (response.statusCode == 200) {
        _showSuccess('Stake successful!');
        _loadData();
      } else {
        _showError('Stake failed');
      }
    } catch (e) {
      _showSuccess('Demo stake executed!');
      _loadData();
    }
  }

  Future<void> _unstake(String token, String amount) async {
    try {
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/staking/unstake'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'token': token,
          'amount': amount,
        }),
      );

      if (response.statusCode == 200) {
        _showSuccess('Unstake initiated!');
        _loadData();
      } else {
        _showError('Unstake failed');
      }
    } catch (e) {
      _showSuccess('Demo unstake executed!');
      _loadData();
    }
  }

  void _showError(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message), backgroundColor: AppColors.error),
    );
  }

  void _showSuccess(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message), backgroundColor: AppColors.success),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('Staking', style: TextStyle(color: AppColors.textPrimary)),
        bottom: TabBar(
          controller: _tabController,
          indicatorColor: AppColors.primary,
          labelColor: AppColors.primary,
          unselectedLabelColor: AppColors.textSecondary,
          tabs: const [
            Tab(text: 'Earn'),
            Tab(text: 'My Stakes'),
            Tab(text: 'Rewards'),
          ],
        ),
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildEarnTab(),
                _buildMyStakesTab(),
                _buildRewardsTab(),
              ],
            ),
    );
  }

  Widget _buildEarnTab() {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _stakingPools.length,
      itemBuilder: (context, index) {
        final pool = _stakingPools[index];
        return _buildStakingPoolCard(pool);
      },
    );
  }

  Widget _buildStakingPoolCard(Map<String, dynamic> pool) {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Row(
                children: [
                  Container(
                    width: 48,
                    height: 48,
                    decoration: BoxDecoration(
                      color: AppColors.primary.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(14),
                    ),
                    child: Center(
                      child: Text(
                        pool['token']?.substring(0, 1) ?? '?',
                        style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          color: AppColors.primary,
                          fontSize: 20,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('${pool['token']} Staking',
                          style: const TextStyle(
                              fontWeight: FontWeight.bold,
                              fontSize: 16,
                              color: AppColors.textPrimary)),
                      Text(pool['chain'],
                          style: TextStyle(
                              color: AppColors.textSecondary, fontSize: 12)),
                    ],
                  ),
                ],
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color: AppColors.success.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.trending_up,
                        color: AppColors.success, size: 16),
                    const SizedBox(width: 4),
                    Text('${pool['apr']}% APR',
                        style: const TextStyle(
                            color: AppColors.success,
                            fontWeight: FontWeight.bold)),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              _buildPoolInfo('Min Stake', pool['min_stake']),
              _buildPoolInfo('Lock Period',
                  pool['lock_period'] == 0 ? 'None' : '${pool['lock_period']} days'),
              _buildPoolInfo('Total Staked', _formatNumber(pool['total_staked'])),
              _buildPoolInfo('Validators', '${pool['validators']}'),
            ],
          ),
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: () => _showStakeDialog(pool),
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.primary,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12)),
              ),
              child: const Text('Stake Now',
                  style: TextStyle(fontWeight: FontWeight.bold)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPoolInfo(String label, String value) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyle(color: AppColors.textSecondary, fontSize: 11)),
        Text(value,
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
      ],
    );
  }

  Widget _buildMyStakesTab() {
    if (_userStakes.isEmpty) {
      return const Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.account_balance_wallet_outlined,
                size: 64, color: AppColors.textSecondary),
            SizedBox(height: 16),
            Text('No active stakes',
                style: TextStyle(color: AppColors.textSecondary)),
            SizedBox(height: 8),
            Text('Start earning by staking your tokens',
                style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
          ],
        ),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _userStakes.length,
      itemBuilder: (context, index) {
        final stake = _userStakes[index];
        return _buildUserStakeCard(stake);
      },
    );
  }

  Widget _buildUserStakeCard(Map<String, dynamic> stake) {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text('${stake['token']} Staking',
                  style: const TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 16,
                      color: AppColors.textPrimary)),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: AppColors.success.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text('${stake['apr']}% APR',
                    style: const TextStyle(
                        color: AppColors.success, fontSize: 12)),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              _buildStakeInfo('Staked', '${stake['amount']} ${stake['token']}'),
              _buildStakeInfo('Value', '\$${stake['value_usd']}'),
              _buildStakeInfo('Rewards', '${stake['rewards']} ${stake['token']}'),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: () => _showUnstakeDialog(stake),
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    side: const BorderSide(color: AppColors.error),
                  ),
                  child: const Text('Unstake',
                      style: TextStyle(color: AppColors.error)),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: () {},
                  style: ElevatedButton.styleFrom(
                    backgroundColor: AppColors.primary,
                    padding: const EdgeInsets.symmetric(vertical: 12),
                  ),
                  child: const Text('Claim Rewards'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildStakeInfo(String label, String value) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
        Text(value,
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
      ],
    );
  }

  Widget _buildRewardsTab() {
    double totalRewards = 0;
    for (var stake in _userStakes) {
      totalRewards += (stake['rewards'] ?? 0).toDouble();
    }

    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.card_giftcard,
              size: 64, color: AppColors.primary),
          const SizedBox(height: 16),
          const Text('Total Rewards',
              style: TextStyle(color: AppColors.textSecondary)),
          const SizedBox(height: 8),
          Text('${totalRewards.toStringAsFixed(6)}',
              style: const TextStyle(
                  fontSize: 32,
                  fontWeight: FontWeight.bold,
                  color: AppColors.primary)),
          const SizedBox(height: 24),
          ElevatedButton(
            onPressed: totalRewards > 0 ? () {} : null,
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.primary,
              padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 14),
            ),
            child: const Text('Claim All Rewards'),
          ),
        ],
      ),
    );
  }

  void _showStakeDialog(Map<String, dynamic> pool) {
    final amountController = TextEditingController();
    
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
          left: 20,
          right: 20,
          top: 20,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Stake ${pool['token']}',
                style: const TextStyle(
                    fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 20),
            TextField(
              controller: amountController,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: 'Amount',
                hintText: 'Min: ${pool['min_stake']}',
                suffixText: pool['token'],
                border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12)),
              ),
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: () {
                  Navigator.pop(context);
                  _stake(pool['token'], amountController.text);
                },
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppColors.primary,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                child: const Text('Stake Now'),
              ),
            ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  void _showUnstakeDialog(Map<String, dynamic> stake) {
    final amountController = TextEditingController();
    
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
          left: 20,
          right: 20,
          top: 20,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Unstake ${stake['token']}',
                style: const TextStyle(
                    fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 20),
            TextField(
              controller: amountController,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: 'Amount',
                hintText: 'Max: ${stake['amount']}',
                suffixText: stake['token'],
                border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12)),
              ),
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: () {
                  Navigator.pop(context);
                  _unstake(stake['token'], amountController.text);
                },
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppColors.error,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                child: const Text('Unstake Now'),
              ),
            ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  String _formatNumber(String number) {
    final num = double.tryParse(number) ?? 0;
    if (num >= 1000000000) {
      return '${(num / 1000000000).toStringAsFixed(1)}B';
    } else if (num >= 1000000) {
      return '${(num / 1000000).toStringAsFixed(1)}M';
    } else if (num >= 1000) {
      return '${(num / 1000).toStringAsFixed(1)}K';
    }
    return number;
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }
}
