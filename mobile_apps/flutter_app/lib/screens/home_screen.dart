import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Home Screen - Main dashboard with portfolio overview
class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;
  bool _isLoading = true;
  double _totalBalance = 0.0;
  String _selectedChain = 'all';
  List<Map<String, dynamic>> _assets = [];
  List<Map<String, dynamic>> _transactions = [];
  List<Map<String, dynamic>> _dApps = [];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 4, vsync: this);
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _isLoading = true);
    try {
      // Load assets from API
      await _loadAssets();
      // Load transactions
      await _loadTransactions();
      // Load popular dApps
      await _loadDApps();
    } catch (e) {
      // Handle error
    }
    setState(() => _isLoading = false);
  }

  Future<void> _loadAssets() async {
    try {
      // Call backend API to get user assets
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/wallet/assets'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _assets = List<Map<String, dynamic>>.from(data['assets'] ?? []);
          _totalBalance = (data['total_balance'] ?? 0).toDouble();
        });
      }
    } catch (e) {
      // Use empty list on error
      setState(() {
        _assets = [];
        _totalBalance = 0.0;
      });
    }
  }

  Future<void> _loadTransactions() async {
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/wallet/transactions'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _transactions = List<Map<String, dynamic>>.from(data['transactions'] ?? []);
        });
      }
    } catch (e) {
      setState(() => _transactions = []);
    }
  }

  Future<void> _loadDApps() async {
    // Popular dApps for the browser
    setState(() {
      _dApps = [
        {'name': 'Uniswap', 'icon': '🦄', 'category': 'DeFi', 'url': 'https://app.uniswap.org'},
        {'name': 'Aave', 'icon': '🦁', 'category': 'Lending', 'url': 'https://app.aave.com'},
        {'name': 'OpenSea', 'icon': '🌊', 'category': 'NFT', 'url': 'https://opensea.io'},
        {'name': 'Compound', 'icon': '💰', 'category': 'Lending', 'url': 'https://app.compound.finance'},
        {'name': 'Curve', 'icon': '📈', 'category': 'DeFi', 'url': 'https://curve.fi'},
        {'name': '1inch', 'icon': '🥇', 'category': 'DEX', 'url': 'https://app.1inch.io'},
        {'name': 'PancakeSwap', 'icon': '🥞', 'category': 'DEX', 'url': 'https://pancakeswap.finance'},
        {'name': 'Raydium', 'icon': '🌟', 'category': 'DEX', 'url': 'https://raydium.io'},
      ];
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            _buildHeader(),
            _buildBalanceCard(),
            _buildQuickActions(),
            _buildTabBar(),
            Expanded(child: _buildTabContent()),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader() {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: AppColors.primary.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(Icons.account_balance_wallet,
                    color: AppColors.primary),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('TigerWallet',
                      style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: AppColors.textPrimary)),
                  Text('Web3 Portfolio',
                      style: TextStyle(
                          fontSize: 12, color: AppColors.textSecondary)),
                ],
              ),
            ],
          ),
          Row(
            children: [
              IconButton(
                icon: const Icon(Icons.notifications_outlined,
                    color: AppColors.textPrimary),
                onPressed: () => Navigator.pushNamed(context, '/notifications'),
              ),
              IconButton(
                icon: const Icon(Icons.settings_outlined,
                    color: AppColors.textPrimary),
                onPressed: () => Navigator.pushNamed(context, '/settings'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildBalanceCard() {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [AppColors.primary, AppColors.primaryDark],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(20),
        boxShadow: [
          BoxShadow(
            color: AppColors.primary.withOpacity(0.3),
            blurRadius: 15,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Total Balance',
              style: TextStyle(color: Colors.white70, fontSize: 14)),
          const SizedBox(height: 8),
          Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text('\$${_totalBalance.toStringAsFixed(2)}',
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 32,
                      fontWeight: FontWeight.bold)),
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Colors.green.withOpacity(0.2),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: const Row(
                  children: [
                    Icon(Icons.arrow_upward, color: Colors.green, size: 14),
                    Text('+5.2%',
                        style: TextStyle(color: Colors.green, fontSize: 12)),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              _buildActionButton(Icons.arrow_upward, 'Send', () {}),
              _buildActionButton(Icons.arrow_downward, 'Receive', () {}),
              _buildActionButton(Icons.swap_horiz, 'Swap', () {}),
              _buildActionButton(Icons.add_circle_outline, 'Buy', () {}),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildActionButton(IconData icon, String label, VoidCallback onTap) {
    return GestureDetector(
      onTap: onTap,
      child: Column(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.2),
              borderRadius: BorderRadius.circular(14),
            ),
            child: Icon(icon, color: Colors.white),
          ),
          const SizedBox(height: 4),
          Text(label,
              style:
                  const TextStyle(color: Colors.white70, fontSize: 11)),
        ],
      ),
    );
  }

  Widget _buildQuickActions() {
    return Container(
      height: 100,
      margin: const EdgeInsets.only(top: 16),
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
        children: [
          _buildQuickActionCard('Staking', '💎', 'Earn rewards'),
          _buildQuickActionCard('NFTs', '🎨', 'View collectibles'),
          _buildQuickActionCard('Bridge', '🌉', 'Cross-chain'),
          _buildQuickActionCard('DApps', '🚀', 'Browse apps'),
          _buildQuickActionCard('History', '📜', 'Transactions'),
        ],
      ),
    );
  }

  Widget _buildQuickActionCard(
      String title, String emoji, String subtitle) {
    return Container(
      width: 100,
      margin: const EdgeInsets.only(right: 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(emoji, style: const TextStyle(fontSize: 24)),
          const SizedBox(height: 8),
          Text(title,
              style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  color: AppColors.textPrimary,
                  fontSize: 13)),
          Text(subtitle,
              style: TextStyle(color: AppColors.textSecondary, fontSize: 10)),
        ],
      ),
    );
  }

  Widget _buildTabBar() {
    return Container(
      margin: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(12),
      ),
      child: TabBar(
        controller: _tabController,
        indicator: BoxDecoration(
          color: AppColors.primary,
          borderRadius: BorderRadius.circular(10),
        ),
        labelColor: Colors.white,
        unselectedLabelColor: AppColors.textSecondary,
        labelStyle: const TextStyle(fontWeight: FontWeight.w600, fontSize: 12),
        tabs: const [
          Tab(text: 'Assets'),
          Tab(text: 'DeFi'),
          Tab(text: 'NFTs'),
          Tab(text: 'History'),
        ],
      ),
    );
  }

  Widget _buildTabContent() {
    return TabBarView(
      controller: _tabController,
      children: [
        _buildAssetsTab(),
        _buildDeFiTab(),
        _buildNFTsTab(),
        _buildHistoryTab(),
      ],
    );
  }

  Widget _buildAssetsTab() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_assets.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.account_balance_wallet_outlined,
                size: 64, color: AppColors.textSecondary),
            const SizedBox(height: 16),
            const Text('No assets yet',
                style: TextStyle(color: AppColors.textSecondary)),
            const SizedBox(height: 8),
            ElevatedButton(
              onPressed: () {},
              child: const Text('Add Token'),
            ),
          ],
        ),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      itemCount: _assets.length,
      itemBuilder: (context, index) {
        final asset = _assets[index];
        return _buildAssetItem(asset);
      },
    );
  }

  Widget _buildAssetItem(Map<String, dynamic> asset) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: AppColors.primary.withOpacity(0.1),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Center(
              child: Text(
                asset['symbol']?.toString().substring(0, 1) ?? '?',
                style: const TextStyle(
                    fontWeight: FontWeight.bold,
                    color: AppColors.primary,
                    fontSize: 18),
              ),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(asset['name'] ?? '',
                    style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        color: AppColors.textPrimary)),
                Text(asset['chain'] ?? '',
                    style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
              ],
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                  '\$${((asset['balance'] ?? 0) * (asset['price'] ?? 0)).toStringAsFixed(2)}',
                  style: const TextStyle(
                      fontWeight: FontWeight.bold,
                      color: AppColors.textPrimary)),
              Text('${asset['balance']} ${asset['symbol']}',
                  style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildDeFiTab() {
    return GridView.builder(
      padding: const EdgeInsets.all(16),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        childAspectRatio: 1.5,
        crossAxisSpacing: 12,
        mainAxisSpacing: 12,
      ),
      itemCount: _dApps.length,
      itemBuilder: (context, index) {
        final dapp = _dApps[index];
        return _buildDAppCard(dapp);
      },
    );
  }

  Widget _buildDAppCard(Map<String, dynamic> dapp) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(dapp['icon'], style: const TextStyle(fontSize: 28)),
          const SizedBox(height: 8),
          Text(dapp['name'],
              style: const TextStyle(
                  fontWeight: FontWeight.w600, color: AppColors.textPrimary)),
          Text(dapp['category'],
              style: TextStyle(color: AppColors.textSecondary, fontSize: 11)),
        ],
      ),
    );
  }

  Widget _buildNFTsTab() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.image_outlined,
              size: 64, color: AppColors.textSecondary),
          const SizedBox(height: 16),
          const Text('No NFTs yet',
              style: TextStyle(color: AppColors.textSecondary)),
          const SizedBox(height: 8),
          ElevatedButton(
            onPressed: () {},
            child: const Text('Explore NFTs'),
          ),
        ],
      ),
    );
  }

  Widget _buildHistoryTab() {
    if (_transactions.isEmpty) {
      return const Center(
        child: Text('No transactions yet',
            style: TextStyle(color: AppColors.textSecondary)),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      itemCount: _transactions.length,
      itemBuilder: (context, index) {
        final tx = _transactions[index];
        return _buildTransactionItem(tx);
      },
    );
  }

  Widget _buildTransactionItem(Map<String, dynamic> tx) {
    final isSend = tx['type'] == 'send';
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: isSend
                  ? Colors.red.withOpacity(0.1)
                  : Colors.green.withOpacity(0.1),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(
              isSend ? Icons.arrow_upward : Icons.arrow_downward,
              color: isSend ? Colors.red : Colors.green,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(isSend ? 'Sent' : 'Received',
                    style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        color: AppColors.textPrimary)),
                Text(tx['date'] ?? '',
                    style: TextStyle(color: AppColors.textSecondary, fontSize: 11)),
              ],
            ),
          ),
          Text(
              '${isSend ? '-' : '+'}${tx['amount']} ${tx['symbol']}',
              style: TextStyle(
                  fontWeight: FontWeight.bold,
                  color: isSend ? Colors.red : Colors.green)),
        ],
      ),
    );
  }
}
