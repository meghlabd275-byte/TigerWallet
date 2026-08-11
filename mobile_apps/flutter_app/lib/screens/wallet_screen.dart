import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';
import 'send_screen.dart';

/// Wallet Screen - View and manage wallet addresses across chains
class WalletScreen extends StatefulWidget {
  const WalletScreen({super.key});

  @override
  State<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends State<WalletScreen> {
  List<Map<String, dynamic>> _wallets = [];
  bool _isLoading = true;
  String _selectedChain = 'all';

  @override
  void initState() {
    super.initState();
    _loadWallets();
  }

  Future<void> _loadWallets() async {
    setState(() => _isLoading = true);
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/wallet/addresses'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _wallets = List<Map<String, dynamic>>.from(data['wallets'] ?? []);
        });
      } else {
        // Honest empty state on backend failure — never show fabricated
        // demo wallets with fake balances/addresses.
        setState(() => _wallets = []);
      }
    } catch (e) {
      // Honest empty state on network error — no demo wallets.
      setState(() => _wallets = []);
    }
    setState(() => _isLoading = false);
  }


  void _copyToClipboard(String text) {
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Address copied to clipboard'),
        backgroundColor: AppColors.success,
        duration: Duration(seconds: 2),
      ),
    );
  }

  String _formatAddress(String address) {
    if (address.length <= 16) return address;
    return '${address.substring(0, 8)}...${address.substring(address.length - 8)}';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('My Wallets',
            style: TextStyle(color: AppColors.textPrimary)),
        actions: [
          IconButton(
            icon: const Icon(Icons.add_circle_outline, color: AppColors.primary),
            onPressed: () => _showAddChainDialog(),
          ),
        ],
      ),
      body: Column(
        children: [
          _buildFilterChips(),
          Expanded(child: _buildWalletList()),
        ],
      ),
    );
  }

  Widget _buildFilterChips() {
    final chains = ['all', 'Ethereum', 'BNB Chain', 'Polygon', 'Solana', 'Avalanche', 'Arbitrum'];
    
    return Container(
      height: 50,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: ListView.builder(
        scrollDirection: Axis.horizontal,
        itemCount: chains.length,
        itemBuilder: (context, index) {
          final chain = chains[index];
          final isSelected = _selectedChain == chain;
          
          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: FilterChip(
              label: Text(chain == 'all' ? 'All Chains' : chain),
              selected: isSelected,
              onSelected: (selected) {
                setState(() => _selectedChain = chain);
              },
              backgroundColor: AppColors.cardBackground,
              selectedColor: AppColors.primary,
              labelStyle: TextStyle(
                color: isSelected ? Colors.white : AppColors.textSecondary,
                fontSize: 12,
              ),
              checkmarkColor: Colors.white,
            ),
          );
        },
      ),
    );
  }

  Widget _buildWalletList() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    final filteredWallets = _selectedChain == 'all'
        ? _wallets
        : _wallets.where((w) => w['chain'] == _selectedChain).toList();

    if (filteredWallets.isEmpty) {
      return const Center(
        child: Text('No wallets found',
            style: TextStyle(color: AppColors.textSecondary)),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: filteredWallets.length,
      itemBuilder: (context, index) {
        final wallet = filteredWallets[index];
        return _buildWalletCard(wallet);
      },
    );
  }

  Widget _buildWalletCard(Map<String, dynamic> wallet) {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
        gradient: LinearGradient(
          colors: [
            AppColors.cardBackground,
            AppColors.primary.withOpacity(0.05),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
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
                    width: 40,
                    height: 40,
                    decoration: BoxDecoration(
                      color: _getChainColor(wallet['chain']).withOpacity(0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Center(
                      child: Text(
                        wallet['symbol']?.substring(0, 1) ?? '?',
                        style: TextStyle(
                          fontWeight: FontWeight.bold,
                          color: _getChainColor(wallet['chain']),
                          fontSize: 18,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(wallet['chain'],
                          style: const TextStyle(
                              fontWeight: FontWeight.w600,
                              color: AppColors.textPrimary,
                              fontSize: 16)),
                      Text(wallet['symbol'],
                          style: TextStyle(
                              color: AppColors.textSecondary, fontSize: 12)),
                    ],
                  ),
                ],
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: AppColors.success.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  '\$${wallet['balance_usd']}',
                  style: const TextStyle(
                      fontWeight: FontWeight.bold, color: AppColors.success),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppColors.background,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    _formatAddress(wallet['address']),
                    style: const TextStyle(
                        fontFamily: 'monospace',
                        color: AppColors.textSecondary,
                        fontSize: 12),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.copy, size: 18),
                  color: AppColors.textSecondary,
                  onPressed: () => _copyToClipboard(wallet['address']),
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(),
                ),
                const SizedBox(width: 8),
                IconButton(
                  icon: const Icon(Icons.qr_code, size: 18),
                  color: AppColors.textSecondary,
                  onPressed: () => _showQRCode(wallet['address']),
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text('${wallet['balance']} ${wallet['symbol']}',
                  style: const TextStyle(
                      fontWeight: FontWeight.w500,
                      color: AppColors.textPrimary)),
              Row(
                children: [
                  _buildActionButton('Send', Icons.arrow_upward, () {
                    Navigator.push(
                      context,
                      MaterialPageRoute(builder: (context) => const SendScreen()),
                    );
                  }),
                  const SizedBox(width: 8),
                  _buildActionButton('Receive', Icons.arrow_downward, () {}),
                ],
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildActionButton(String label, IconData icon, VoidCallback onTap) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          color: AppColors.primary.withOpacity(0.1),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(icon, size: 16, color: AppColors.primary),
            const SizedBox(width: 4),
            Text(label,
                style: const TextStyle(
                    color: AppColors.primary, fontWeight: FontWeight.w500)),
          ],
        ),
      ),
    );
  }

  Color _getChainColor(String chain) {
    switch (chain) {
      case 'Ethereum':
        return AppColors.primary;
      case 'BNB Chain':
        return Colors.yellow;
      case 'Polygon':
        return Colors.purple;
      case 'Solana':
        return Colors.orange;
      case 'Avalanche':
        return Colors.red;
      case 'Arbitrum':
        return Colors.blue;
      default:
        return AppColors.primary;
    }
  }

  void _showAddChainDialog() {
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Container(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Add Blockchain',
                style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: AppColors.textPrimary)),
            const SizedBox(height: 16),
            const Text('Select a blockchain to add wallet address:',
                style: TextStyle(color: AppColors.textSecondary)),
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                _buildChainChip('Ethereum', 1),
                _buildChainChip('BNB Chain', 56),
                _buildChainChip('Polygon', 137),
                _buildChainChip('Solana', 101),
                _buildChainChip('Avalanche', 43114),
                _buildChainChip('Arbitrum', 42161),
                _buildChainChip('Optimism', 10),
                _buildChainChip('Base', 8453),
              ],
            ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  Widget _buildChainChip(String chain, int chainId) {
    return ActionChip(
      label: Text(chain),
      onPressed: () {
        Navigator.pop(context);
        _addChainWallet(chain, chainId);
      },
      backgroundColor: AppColors.background,
    );
  }

  Future<void> _addChainWallet(String chain, int chainId) async {
    // Show loading
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (context) => const Center(child: CircularProgressIndicator()),
    );

    try {
      // Call API to generate wallet address for new chain
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/wallet/generate'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({'chain_id': chainId}),
      );

      Navigator.pop(context); // Close loading

      if (response.statusCode == 200) {
        _loadWallets(); // Reload wallets
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('$chain wallet added successfully'),
            backgroundColor: AppColors.success,
          ),
        );
      }
    } catch (e) {
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to add wallet: $e'),
          backgroundColor: AppColors.error,
        ),
      );
    }
  }

  void _showQRCode(String address) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: AppColors.cardBackground,
        title: const Text('Scan QR Code',
            style: TextStyle(color: AppColors.textPrimary)),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 200,
              height: 200,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
              ),
              child: const Center(
                child: Icon(Icons.qr_code_2, size: 150, color: Colors.black),
              ),
            ),
            const SizedBox(height: 16),
            SelectableText(address,
                style: const TextStyle(
                    fontFamily: 'monospace', fontSize: 10)),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }
}
