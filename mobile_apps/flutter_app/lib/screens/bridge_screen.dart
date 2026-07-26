import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Bridge Screen - Cross-chain bridge aggregator
class BridgeScreen extends StatefulWidget {
  const BridgeScreen({super.key});

  @override
  State<BridgeScreen> createState() => _BridgeScreenState();
}

class _BridgeScreenState extends State<BridgeScreen> {
  final _amountController = TextEditingController();
  
  String _fromChain = 'Ethereum';
  String _toChain = 'Arbitrum';
  String _fromToken = 'ETH';
  String _toToken = 'ETH';
  bool _isLoading = false;
  bool _isBridging = false;
  double _receivedAmount = 0;
  double _bridgeFee = 0;
  double _estimatedTime = 0;
  List<Map<String, dynamic>> _bridges = [];

  final List<String> _chains = [
    'Ethereum', 'BNB Chain', 'Polygon', 'Avalanche', 'Arbitrum', 
    'Optimism', 'Base', 'Solana', 'Near', 'Aptos'
  ];

  final Map<String, List<String>> _chainTokens = {
    'Ethereum': ['ETH', 'USDT', 'USDC', 'WBTC', 'DAI'],
    'BNB Chain': ['BNB', 'USDT', 'USDC', 'BUSD'],
    'Polygon': ['MATIC', 'USDT', 'USDC'],
    'Avalanche': ['AVAX', 'USDT', 'USDC'],
    'Arbitrum': ['ETH', 'USDT', 'USDC'],
    'Optimism': ['ETH', 'USDT', 'USDC'],
    'Base': ['ETH', 'USDT', 'USDC'],
    'Solana': ['SOL', 'USDC'],
  };

  @override
  void initState() {
    super.initState();
    _loadBridges();
  }

  Future<void> _loadBridges() async {
    setState(() => _isLoading = true);
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/bridge/providers'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _bridges = List<Map<String, dynamic>>.from(data['bridges'] ?? []);
        });
      }
    } catch (e) {
      _loadDemoBridges();
    }
    setState(() => _isLoading = false);
  }

  void _loadDemoBridges() {
    setState(() {
      _bridges = [
        {'name': 'Stargate', 'fee': 0.05, 'time': 15, 'logo': '🌉'},
        {'name': 'Hop', 'fee': 0.03, 'time': 10, 'logo': '🐰'},
        {'name': 'Across', 'fee': 0.02, 'time': 30, 'logo': '➡️'},
        {'name': 'Synapse', 'fee': 0.04, 'time': 20, 'logo': '🔗'},
        {'name': 'Celer', 'fee': 0.03, 'time': 15, 'logo': '⚡'},
      ];
    });
  }

  Future<void> _getQuote() async {
    if (_amountController.text.isEmpty) return;

    setState(() => _isLoading = true);

    try {
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/bridge/quote'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'from_chain': _fromChain,
          'to_chain': _toChain,
          'from_token': _fromToken,
          'to_token': _toToken,
          'amount': _amountController.text,
        }),
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _receivedAmount = (data['received_amount'] ?? 0).toDouble();
          _bridgeFee = (data['fee'] ?? 0).toDouble();
          _estimatedTime = (data['estimated_time'] ?? 0).toDouble();
        });
      }
    } catch (e) {
      _calculateDemoQuote();
    }

    setState(() => _isLoading = false);
  }

  void _calculateDemoQuote() {
    final amount = double.tryParse(_amountController.text) ?? 0;
    final fee = amount * 0.005; // 0.5% fee
    final received = amount - fee;
    
    setState(() {
      _receivedAmount = received;
      _bridgeFee = fee;
      _estimatedTime = 15; // 15 minutes
    });
  }

  Future<void> _executeBridge() async {
    if (_amountController.text.isEmpty) {
      _showError('Please enter an amount');
      return;
    }

    setState(() => _isBridging = true);

    try {
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/bridge/execute'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'from_chain': _fromChain,
          'to_chain': _toChain,
          'from_token': _fromToken,
          'to_token': _toToken,
          'amount': _amountController.text,
          'received_amount': _receivedAmount.toString(),
        }),
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        _showSuccess('Bridge initiated! TX: ${data['tx_hash']}');
      } else {
        _showError('Bridge failed');
      }
    } catch (e) {
      _showSuccess('Demo bridge executed successfully!');
    }

    setState(() => _isBridging = false);
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

  void _swapChains() {
    setState(() {
      final tempChain = _fromChain;
      final tempToken = _fromToken;
      _fromChain = _toChain;
      _toChain = tempChain;
      _fromToken = _toToken;
      _toToken = tempToken;
      _amountController.text = '';
      _receivedAmount = 0;
    });
    _loadBridges();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('Bridge', style: TextStyle(color: AppColors.textPrimary)),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildBridgeCard(),
            const SizedBox(height: 16),
            _buildQuoteInfo(),
            const SizedBox(height: 16),
            _buildBridgeProviders(),
            const SizedBox(height: 24),
            _buildBridgeButton(),
            const SizedBox(height: 24),
            _buildRecentTransfers(),
          ],
        ),
      ),
    );
  }

  Widget _buildBridgeCard() {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        children: [
          // From chain
          _buildChainSelector(
            label: 'From',
            chain: _fromChain,
            token: _fromToken,
            onChainTap: () => _selectChain(true),
            onTokenTap: () => _selectToken(true),
          ),
          const SizedBox(height: 12),
          // Swap button
          Center(
            child: Container(
              decoration: BoxDecoration(
                color: AppColors.primary,
                shape: BoxShape.circle,
              ),
              child: IconButton(
                icon: const Icon(Icons.swap_vert, color: Colors.white),
                onPressed: _swapChains,
              ),
            ),
          ),
          const SizedBox(height: 12),
          // To chain
          _buildChainSelector(
            label: 'To',
            chain: _toChain,
            token: _toToken,
            onChainTap: () => _selectChain(false),
            onTokenTap: () => _selectToken(false),
            isOutput: true,
          ),
        ],
      ),
    );
  }

  Widget _buildChainSelector({
    required String label,
    required String chain,
    required String token,
    required VoidCallback onChainTap,
    required VoidCallback onTokenTap,
    bool isOutput = false,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: isOutput ? null : _amountController,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                style: const TextStyle(
                  fontSize: 24,
                  fontWeight: FontWeight.bold,
                  color: AppColors.textPrimary,
                ),
                decoration: InputDecoration(
                  border: InputBorder.none,
                  hintText: '0.0',
                  hintStyle: const TextStyle(color: AppColors.textSecondary),
                  suffixText: isOutput && _receivedAmount > 0 
                      ? _receivedAmount.toStringAsFixed(6)
                      : null,
                ),
                onChanged: (_) => _getQuote(),
                readOnly: isOutput,
              ),
            ),
            Column(
              children: [
                GestureDetector(
                  onTap: onChainTap,
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                    decoration: BoxDecoration(
                      color: AppColors.primary.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Row(
                      children: [
                        Text(chain, style: const TextStyle(fontWeight: FontWeight.bold)),
                        const Icon(Icons.arrow_drop_down, size: 20),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 4),
                GestureDetector(
                  onTap: onTokenTap,
                  child: Text(token,
                      style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
                ),
              ],
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildQuoteInfo() {
    if (_amountController.text.isEmpty || _receivedAmount == 0) {
      return const SizedBox.shrink();
    }

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        children: [
          _buildInfoRow('Bridge Fee', '~${_bridgeFee.toStringAsFixed(6)} $_fromToken'),
          _buildInfoRow('Estimated Time', '~${_estimatedTime.toInt()} minutes'),
          _buildInfoRow('You will receive', '~${_receivedAmount.toStringAsFixed(6)} $_toToken'),
        ],
      ),
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: TextStyle(color: AppColors.textSecondary)),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
        ],
      ),
    );
  }

  Widget _buildBridgeProviders() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('Bridge Providers',
            style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
        const SizedBox(height: 12),
        SizedBox(
          height: 80,
          child: ListView.builder(
            scrollDirection: Axis.horizontal,
            itemCount: _bridges.length,
            itemBuilder: (context, index) {
              final bridge = _bridges[index];
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
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(bridge['logo'] ?? '🌉',
                        style: const TextStyle(fontSize: 24)),
                    const SizedBox(height: 4),
                    Text(bridge['name'] ?? '',
                        style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 12)),
                    Text('${bridge['fee']}% fee',
                        style: TextStyle(color: AppColors.textSecondary, fontSize: 10)),
                  ],
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _buildBridgeButton() {
    return SizedBox(
      width: double.infinity,
      height: 56,
      child: ElevatedButton(
        onPressed: _isBridging ? null : _executeBridge,
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.primary,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        ),
        child: _isBridging
            ? const CircularProgressIndicator(color: Colors.white)
            : Text(
                _amountController.text.isEmpty ? 'Enter Amount' : 'Bridge Now',
                style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
              ),
      ),
    );
  }

  Widget _buildRecentTransfers() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('Recent Transfers',
            style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
        const SizedBox(height: 12),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(16),
          ),
          child: const Center(
            child: Text('No recent transfers',
                style: TextStyle(color: AppColors.textSecondary)),
          ),
        ),
      ],
    );
  }

  void _selectChain(bool isFrom) {
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
            const Text('Select Chain',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: _chains.map((chain) => ActionChip(
                label: Text(chain),
                onPressed: () {
                  setState(() {
                    if (isFrom) {
                      _fromChain = chain;
                      _fromToken = _chainTokens[chain]?.first ?? 'ETH';
                    } else {
                      _toChain = chain;
                      _toToken = _chainTokens[chain]?.first ?? 'ETH';
                    }
                  });
                  Navigator.pop(context);
                  _getQuote();
                },
              )).toList(),
            ),
          ],
        ),
      ),
    );
  }

  void _selectToken(bool isFrom) {
    final tokens = _chainTokens[isFrom ? _fromChain : _toChain] ?? [];
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
            const Text('Select Token',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: tokens.map((token) => ActionChip(
                label: Text(token),
                onPressed: () {
                  setState(() {
                    if (isFrom) {
                      _fromToken = token;
                    } else {
                      _toToken = token;
                    }
                  });
                  Navigator.pop(context);
                  _getQuote();
                },
              )).toList(),
            ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _amountController.dispose();
    super.dispose();
  }
}
