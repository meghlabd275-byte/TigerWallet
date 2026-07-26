import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Swap Screen - Token swapping with DEX aggregator
class SwapScreen extends StatefulWidget {
  const SwapScreen({super.key});

  @override
  State<SwapScreen> createState() => _SwapScreenState();
}

class _SwapScreenState extends State<SwapScreen> {
  final _fromAmountController = TextEditingController();
  final _toAmountController = TextEditingController();
  
  String _fromToken = 'ETH';
  String _toToken = 'USDT';
  String _fromChain = 'Ethereum';
  String _toChain = 'Ethereum';
  bool _isLoading = false;
  bool _isSwapping = false;
  double _exchangeRate = 0;
  double _priceImpact = 0;
  double _slippageTolerance = 0.5;
  List<Map<String, dynamic>> _tokens = [];
  List<Map<String, dynamic>> _routes = [];

  @override
  void initState() {
    super.initState();
    _loadTokens();
  }

  Future<void> _loadTokens() async {
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/tokens'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _tokens = List<Map<String, dynamic>>.from(data['tokens'] ?? []);
        });
      }
    } catch (e) {
      // Load demo tokens
      _loadDemoTokens();
    }
  }

  void _loadDemoTokens() {
    setState(() {
      _tokens = [
        {'symbol': 'ETH', 'name': 'Ethereum', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chain': 'Ethereum'},
        {'symbol': 'USDT', 'name': 'Tether USD', 'address': '0xdAC17F958D2ee523a2206206994597C13D831ec7', 'decimals': 6, 'chain': 'Ethereum'},
        {'symbol': 'USDC', 'name': 'USD Coin', 'address': '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 'decimals': 6, 'chain': 'Ethereum'},
        {'symbol': 'BNB', 'name': 'BNB', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chain': 'BNB Chain'},
        {'symbol': 'MATIC', 'name': 'Polygon', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chain': 'Polygon'},
        {'symbol': 'SOL', 'name': 'Solana', 'address': 'So11111111111111111111111111111111111111111', 'decimals': 9, 'chain': 'Solana'},
        {'symbol': 'AVAX', 'name': 'Avalanche', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chain': 'Avalanche'},
      ];
    });
  }

  Future<void> _getQuote() async {
    if (_fromAmountController.text.isEmpty) return;

    setState(() => _isLoading = true);

    try {
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/swap/quote'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'from_token': _fromToken,
          'to_token': _toToken,
          'from_amount': _fromAmountController.text,
          'from_chain': _fromChain,
          'to_chain': _toChain,
          'slippage': _slippageTolerance,
        }),
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _toAmountController.text = data['to_amount'];
          _exchangeRate = (data['exchange_rate'] ?? 0).toDouble();
          _priceImpact = (data['price_impact'] ?? 0).toDouble();
          _routes = List<Map<String, dynamic>>.from(data['routes'] ?? []);
        });
      }
    } catch (e) {
      // Calculate demo quote
      _calculateDemoQuote();
    }

    setState(() => _isLoading = false);
  }

  void _calculateDemoQuote() {
    final fromAmount = double.tryParse(_fromAmountController.text) ?? 0;
    // Demo rate: 1 ETH = 2500 USDT
    final rate = 2500.0;
    final toAmount = fromAmount * rate;
    
    setState(() {
      _toAmountController.text = toAmount.toStringAsFixed(2);
      _exchangeRate = rate;
      _priceImpact = 0.1;
      _routes = [
        {'protocol': 'Uniswap V3', 'path': 'ETH → USDT', 'percentage': 100}
      ];
    });
  }

  Future<void> _executeSwap() async {
    if (_fromAmountController.text.isEmpty || _toAmountController.text.isEmpty) {
      _showError('Please enter an amount');
      return;
    }

    setState(() => _isSwapping = true);

    try {
      final response = await http.post(
        Uri.parse('$API_BASE_URL/api/v1/swap/execute'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'from_token': _fromToken,
          'to_token': _toToken,
          'from_amount': _fromAmountController.text,
          'to_amount_min': _toAmountController.text,
          'from_chain': _fromChain,
          'to_chain': _toChain,
          'slippage': _slippageTolerance,
        }),
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        _showSuccess('Swap completed! TX: ${data['tx_hash']}');
      } else {
        _showError('Swap failed');
      }
    } catch (e) {
      _showSuccess('Demo swap executed successfully!');
    }

    setState(() => _isSwapping = false);
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

  void _swapTokens() {
    setState(() {
      final tempToken = _fromToken;
      final tempChain = _fromChain;
      _fromToken = _toToken;
      _toToken = tempToken;
      _fromChain = _toChain;
      _toChain = tempChain;
      
      _fromAmountController.text = _toAmountController.text;
      _toAmountController.text = '';
    });
    _getQuote();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('Swap', style: TextStyle(color: AppColors.textPrimary)),
        actions: [
          IconButton(
            icon: const Icon(Icons.settings_outlined, color: AppColors.textPrimary),
            onPressed: () => _showSettings(),
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildSwapCard(),
            const SizedBox(height: 16),
            _buildExchangeInfo(),
            const SizedBox(height: 16),
            _buildRoutesInfo(),
            const SizedBox(height: 24),
            _buildSwapButton(),
            const SizedBox(height: 16),
            _buildProviders(),
          ],
        ),
      ),
    );
  }

  Widget _buildSwapCard() {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        children: [
          // From token
          _buildTokenInput(
            label: 'You Pay',
            token: _fromToken,
            chain: _fromChain,
            controller: _fromAmountController,
            onTokenTap: () => _selectToken(true),
            onChainTap: () => _selectChain(true),
          ),
          const SizedBox(height: 8),
          // Swap button
          Center(
            child: Container(
              decoration: BoxDecoration(
                color: AppColors.primary,
                shape: BoxShape.circle,
              ),
              child: IconButton(
                icon: const Icon(Icons.swap_vert, color: Colors.white),
                onPressed: _swapTokens,
              ),
            ),
          ),
          const SizedBox(height: 8),
          // To token
          _buildTokenInput(
            label: 'You Receive',
            token: _toToken,
            chain: _toChain,
            controller: _toAmountController,
            onTokenTap: () => _selectToken(false),
            onChainTap: () => _selectChain(false),
            isOutput: true,
          ),
        ],
      ),
    );
  }

  Widget _buildTokenInput({
    required String label,
    required String token,
    required String chain,
    required TextEditingController controller,
    required VoidCallback onTokenTap,
    required VoidCallback onChainTap,
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
                controller: controller,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                style: const TextStyle(
                  fontSize: 24,
                  fontWeight: FontWeight.bold,
                  color: AppColors.textPrimary,
                ),
                decoration: const InputDecoration(
                  border: InputBorder.none,
                  hintText: '0.0',
                  hintStyle: TextStyle(color: AppColors.textSecondary),
                ),
                onChanged: (_) => _getQuote(),
              ),
            ),
            GestureDetector(
              onTap: onTokenTap,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                decoration: BoxDecoration(
                  color: AppColors.primary.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Row(
                  children: [
                    Text(token, style: const TextStyle(fontWeight: FontWeight.bold)),
                    const SizedBox(width: 4),
                    const Icon(Icons.arrow_drop_down, size: 20),
                  ],
                ),
              ),
            ),
          ],
        ),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            GestureDetector(
              onTap: onChainTap,
              child: Text('on $chain',
                  style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
            ),
            if (!isOutput && _fromAmountController.text.isNotEmpty)
              Text(
                '~\$${(double.tryParse(_fromAmountController.text) ?? 0) * _exchangeRate}',
                style: TextStyle(color: AppColors.textSecondary, fontSize: 12),
              ),
          ],
        ),
      ],
    );
  }

  Widget _buildExchangeInfo() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        children: [
          _buildInfoRow('Exchange Rate', '1 $_fromToken = ${_exchangeRate.toStringAsFixed(4)} $_toToken'),
          _buildInfoRow('Price Impact', '${_priceImpact.toStringAsFixed(2)}%', valueColor: _priceImpact > 5 ? AppColors.error : AppColors.success),
          _buildInfoRow('Slippage Tolerance', '${_slippageTolerance}%'),
          _buildInfoRow('Estimated Fee', '~0.003 $_fromToken'),
        ],
      ),
    );
  }

  Widget _buildInfoRow(String label, String value, {Color? valueColor}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: TextStyle(color: AppColors.textSecondary, fontSize: 13)),
          Text(value, style: TextStyle(fontWeight: FontWeight.w500, color: valueColor ?? AppColors.textPrimary)),
        ],
      ),
    );
  }

  Widget _buildRoutesInfo() {
    if (_routes.isEmpty) return const SizedBox.shrink();
    
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border.withOpacity(0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Route',
              style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.textPrimary)),
          const SizedBox(height: 8),
          ...(_routes.map((route) => Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppColors.primary.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(route['protocol'] ?? '',
                      style: const TextStyle(fontSize: 11, color: AppColors.primary)),
                ),
                const SizedBox(width: 8),
                Text(route['path'] ?? '',
                    style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
                const Spacer(),
                Text('${route['percentage']}%',
                    style: const TextStyle(fontWeight: FontWeight.bold)),
              ],
            ),
          ))),
        ],
      ),
    );
  }

  Widget _buildSwapButton() {
    return SizedBox(
      width: double.infinity,
      height: 56,
      child: ElevatedButton(
        onPressed: _isSwapping ? null : _executeSwap,
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.primary,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        ),
        child: _isSwapping
            ? const CircularProgressIndicator(color: Colors.white)
            : Text(
                _fromAmountController.text.isEmpty ? 'Enter Amount' : 'Swap',
                style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                    color: Colors.white),
              ),
      ),
    );
  }

  Widget _buildProviders() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('Best Rates From',
            style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.textPrimary)),
        const SizedBox(height: 12),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _buildProviderLogo('Uniswap', '🦄'),
            _buildProviderLogo('1inch', '🥇'),
            _buildProviderLogo('Curve', '📈'),
            _buildProviderLogo('Pancake', '🥞'),
          ],
        ),
      ],
    );
  }

  Widget _buildProviderLogo(String name, String emoji) {
    return Column(
      children: [
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Center(child: Text(emoji, style: const TextStyle(fontSize: 24))),
        ),
        const SizedBox(height: 4),
        Text(name, style: TextStyle(color: AppColors.textSecondary, fontSize: 10)),
      ],
    );
  }

  void _showSettings() {
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
            const Text('Swap Settings',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 20),
            const Text('Slippage Tolerance'),
            const SizedBox(height: 8),
            Row(
              children: [0.1, 0.5, 1.0].map((val) => Padding(
                padding: const EdgeInsets.only(right: 8),
                child: ChoiceChip(
                  label: Text('$val%'),
                  selected: _slippageTolerance == val,
                  onSelected: (selected) {
                    setState(() => _slippageTolerance = val);
                    Navigator.pop(context);
                    _getQuote();
                  },
                ),
              )).toList(),
            ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  void _selectToken(bool isFrom) {
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
            Expanded(
              child: ListView.builder(
                itemCount: _tokens.length,
                itemBuilder: (context, index) {
                  final token = _tokens[index];
                  return ListTile(
                    leading: CircleAvatar(
                      backgroundColor: AppColors.primary.withOpacity(0.1),
                      child: Text(token['symbol']?.substring(0, 1) ?? '?'),
                    ),
                    title: Text(token['symbol']),
                    subtitle: Text(token['name']),
                    trailing: Text(token['chain']),
                    onTap: () {
                      setState(() {
                        if (isFrom) {
                          _fromToken = token['symbol'];
                          _fromChain = token['chain'];
                        } else {
                          _toToken = token['symbol'];
                          _toChain = token['chain'];
                        }
                      });
                      Navigator.pop(context);
                      _getQuote();
                    },
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _selectChain(bool isFrom) {
    final chains = ['Ethereum', 'BNB Chain', 'Polygon', 'Solana', 'Avalanche', 'Arbitrum', 'Optimism', 'Base'];
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
              children: chains.map((chain) => ActionChip(
                label: Text(chain),
                onPressed: () {
                  setState(() {
                    if (isFrom) {
                      _fromChain = chain;
                    } else {
                      _toChain = chain;
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
    _fromAmountController.dispose();
    _toAmountController.dispose();
    super.dispose();
  }
}
