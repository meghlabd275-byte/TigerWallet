import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/wallet_service.dart';
import '../utils/theme.dart';

/// Send Screen - Send crypto with QR scanner support
class SendScreen extends StatefulWidget {
  const SendScreen({super.key});

  @override
  State<SendScreen> createState() => _SendScreenState();
}

class _SendScreenState extends State<SendScreen> {
  final _recipientController = TextEditingController();
  final _amountController = TextEditingController();
  final _passwordController = TextEditingController();
  final _secureStorage = const FlutterSecureStorage();
  final _formKey = GlobalKey<FormState>();
  final _walletService = WalletService(
    secureStorage: FlutterSecureStorage(),
  );

  String _selectedChain = 'Ethereum';
  String _selectedToken = 'ETH';
  bool _isLoading = false;
  String? _transactionHash;
  String? _errorMessage;
  bool _obscurePassword = true;

  // R5: auto-approval result surfaced in the success dialog.
  bool? _autoApproved;
  String? _autoApprovalReason;
  
  // Recent addresses are loaded from the backend (transaction history) —
  // no hardcoded demo addresses.
  final List<String> _recentAddresses = [];

  final List<Map<String, dynamic>> _chains = [
    {'id': 'ethereum', 'name': 'Ethereum', 'symbol': 'ETH', 'icon': '🔷'},
    {'id': 'bsc', 'name': 'BNB Chain', 'symbol': 'BNB', 'icon': '🟡'},
    {'id': 'polygon', 'name': 'Polygon', 'symbol': 'MATIC', 'icon': '🟣'},
    {'id': 'arbitrum', 'name': 'Arbitrum', 'symbol': 'ETH', 'icon': '🔵'},
    {'id': 'optimism', 'name': 'Optimism', 'symbol': 'ETH', 'icon': '🔴'},
    {'id': 'avalanche', 'name': 'Avalanche', 'symbol': 'AVAX', 'icon': '❄️'},
    {'id': 'solana', 'name': 'Solana', 'symbol': 'SOL', 'icon': '☀️'},
    {'id': 'tron', 'name': 'TRON', 'symbol': 'TRX', 'icon': '🔺'},
  ];

  final List<String> _tokens = ['ETH', 'USDT', 'USDC', 'BNB', 'MATIC', 'SOL', 'TRX'];

  @override
  void dispose() {
    _recipientController.dispose();
    _amountController.dispose();
    super.dispose();
  }

  // Validate address format
  bool _isValidAddress(String address) {
    // Ethereum
    if (RegExp(r'^0x[a-fA-F0-9]{40}$').hasMatch(address)) return true;
    // Bitcoin
    if (RegExp(r'^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,62}$').hasMatch(address)) return true;
    // Solana
    if (RegExp(r'^[1-9A-HJ-NP-Z]{32,44}$').hasMatch(address)) return true;
    // TRON
    if (RegExp(r'^T[a-zA-HJ-NP-Z0-9]{33}$').hasMatch(address)) return true;
    return false;
  }

  // QR Scanner simulation - in production would use camera package
  void _openQRScanner() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (context) => Container(
        height: MediaQuery.of(context).size.height * 0.7,
        decoration: BoxDecoration(
          color: Theme.of(context).scaffoldBackgroundColor,
          borderRadius: const BorderRadius.vertical(top: Radius.circular(20)),
        ),
        child: Column(
          children: [
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                border: Border(
                  bottom: BorderSide(
                    color: Theme.of(context).dividerColor,
                  ),
                ),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    'Scan QR Code',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.pop(context),
                  ),
                ],
              ),
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  children: [
                    // Camera placeholder
                    Container(
                      height: 250,
                      decoration: BoxDecoration(
                        color: Colors.black,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: Center(
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            const Icon(
                              Icons.qr_code_scanner,
                              size: 80,
                              color: Colors.white54,
                            ),
                            const SizedBox(height: 16),
                            const Text(
                              'Camera QR Scanner',
                              style: TextStyle(color: Colors.white70),
                            ),
                            const SizedBox(height: 8),
                            const Text(
                              'Point camera at QR code',
                              style: TextStyle(color: Colors.white38, fontSize: 12),
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 24),
                    const Text(
                      'OR',
                      style: TextStyle(color: Colors.grey),
                    ),
                    const SizedBox(height: 16),
                    // Recent addresses
                    Text(
                      'Recent Addresses',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 12),
                    Expanded(
                      child: ListView.builder(
                        itemCount: _recentAddresses.length,
                        itemBuilder: (context, index) {
                          final address = _recentAddresses[index];
                          return Card(
                            margin: const EdgeInsets.only(bottom: 8),
                            child: ListTile(
                              title: Text(
                                '${address.substring(0, 10)}...${address.substring(address.length - 8)}',
                                style: const TextStyle(fontFamily: 'monospace'),
                              ),
                              trailing: const Icon(Icons.content_copy),
                              onTap: () {
                                _recipientController.text = address;
                                Navigator.pop(context);
                              },
                            ),
                          );
                        },
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _sendTransaction() async {
    if (!_formKey.currentState!.validate()) return;
    
    final recipient = _recipientController.text.trim();
    final amount = _amountController.text.trim();
    
    if (!_isValidAddress(recipient)) {
      setState(() {
        _errorMessage = 'Invalid recipient address format';
      });
      return;
    }

    setState(() {
      _isLoading = true;
      _errorMessage = null;
      _transactionHash = null;
      _autoApproved = null;
      _autoApprovalReason = null;
    });

    try {
      // R5: Auto sign + approval. Primary path is /auto-send, which performs a
      // server-side policy check and tags the response with auto_approved +
      // auto_approval_reason. The tx is self-signed + broadcast by the backend
      // regardless (self-custody). /send is kept as a fallback when /auto-send
      // is unavailable (e.g. older backend) — in that case there is no
      // auto-approval signal.
      final chainIdMap = <String, int>{
        'ethereum': 1,
        'bsc': 56,
        'polygon': 137,
        'arbitrum': 42161,
        'optimism': 10,
        'avalanche': 43114,
      };
      final chainId = chainIdMap[_selectedChain.toLowerCase()] ?? 1;

      final walletId = await _walletService.getBackendWalletId();
      if (walletId == null || walletId.isEmpty) {
        setState(() {
          _errorMessage =
              'No wallet registered with the backend. Create or import a '
              'wallet first.';
          _isLoading = false;
        });
        return;
      }

      final password = _passwordController.text;
      // unlock_token (passwordless send) takes precedence over password when
      // present; the backend resolves the seed from either.
      final unlockToken =
          await _secureStorage.read(key: 'unlock_token') ?? '';

      String? txHash;
      bool autoApproved = false;
      String? autoApprovalReason;
      String? autoSendError;

      try {
        final result = await _walletService.autoSendTransaction(
          walletId: walletId,
          to: recipient,
          value: amount,
          password: password,
          chainId: chainId,
          gasLimit: '21000',
          unlockToken: (unlockToken.isNotEmpty) ? unlockToken : null,
        );
        txHash = (result['tx_hash'] ?? '').toString();
        autoApproved = result['auto_approved'] == true;
        autoApprovalReason = (result['auto_approval_reason'] ?? '').toString();
        if (autoApprovalReason.isEmpty) autoApprovalReason = null;
      } catch (e) {
        autoSendError = e.toString();
      }

      // Fallback to manual /send when /auto-send was unavailable (non-200,
      // network, or backend without the route). No auto-approval signal here.
      if (txHash == null || txHash.isEmpty) {
        final manualHash = await _walletService.sendTransactionManual(
          walletId: walletId,
          to: recipient,
          value: amount,
          password: password,
          chainId: chainId,
          gasLimit: '21000',
          unlockToken: (unlockToken.isNotEmpty) ? unlockToken : null,
        );
        txHash = manualHash;
        autoApproved = false;
        autoApprovalReason = autoSendError != null
            ? 'Auto-send unavailable — submitted via manual sign. ($autoSendError)'
            : null;
      }

      if (txHash == null || txHash.isEmpty) {
        setState(() {
          _errorMessage = autoSendError ??
              'Backend did not return a transaction hash';
          _isLoading = false;
        });
        return;
      }

      setState(() {
        _transactionHash = txHash;
        _autoApproved = autoApproved;
        _autoApprovalReason = autoApprovalReason;
        _isLoading = false;
      });

      if (mounted) {
        showDialog(
          context: context,
          builder: (context) => AlertDialog(
            title: const Text('Transaction Sent'),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // R4: EXACT required phrase.
                const Text(
                    'Transaction submitted to the blockchain network'),
                const SizedBox(height: 12),
                // R5: ⚡ Auto-approved badge / reason.
                if (autoApproved) ...[
                  Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 10, vertical: 6),
                    decoration: BoxDecoration(
                      color: Colors.amber.withOpacity(0.15),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(
                          color: Colors.amber.withOpacity(0.4)),
                    ),
                    child: const Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text('⚡', style: TextStyle(fontSize: 16)),
                        SizedBox(width: 6),
                        Text(
                          'Auto-approved',
                          style: TextStyle(
                            color: Colors.orange,
                            fontWeight: FontWeight.bold,
                            fontSize: 13,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                ] else if (autoApprovalReason != null &&
                    autoApprovalReason.isNotEmpty) ...[
                  Container(
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: Colors.blueGrey.withOpacity(0.08),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Icon(Icons.bolt, size: 18, color: Colors.blueGrey),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            autoApprovalReason!,
                            style: const TextStyle(fontSize: 12),
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                SelectableText(
                  'Hash: $txHash',
                  style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                ),
              ],
            ),
            actions: [
              TextButton(
                onPressed: () {
                  Navigator.pop(context);
                  Navigator.pop(context);
                },
                child: const Text('OK'),
              ),
            ],
          ),
        );
      }
    } catch (e) {
      setState(() {
        _errorMessage = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Send'),
        actions: [
          IconButton(
            icon: const Icon(Icons.qr_code_scanner),
            onPressed: _openQRScanner,
            tooltip: 'Scan QR Code',
          ),
        ],
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            // Chain Selector
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Select Network',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 12),
                    SizedBox(
                      height: 50,
                      child: ListView.builder(
                        scrollDirection: Axis.horizontal,
                        itemCount: _chains.length,
                        itemBuilder: (context, index) {
                          final chain = _chains[index];
                          final isSelected = _selectedChain == chain['id'];
                          return Padding(
                            padding: const EdgeInsets.only(right: 8),
                            child: ChoiceChip(
                              label: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Text(chain['icon']),
                                  const SizedBox(width: 4),
                                  Text(chain['symbol']),
                                ],
                              ),
                              selected: isSelected,
                              onSelected: (selected) {
                                if (selected) {
                                  setState(() {
                                    _selectedChain = chain['id'];
                                    _selectedToken = chain['symbol'];
                                  });
                                }
                              },
                            ),
                          );
                        },
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            
            // Recipient Address
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          'Recipient Address',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        IconButton(
                          icon: const Icon(Icons.qr_code_scanner),
                          onPressed: _openQRScanner,
                          tooltip: 'Scan QR',
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _recipientController,
                      decoration: InputDecoration(
                        hintText: '0x... or tap QR icon to scan',
                        prefixIcon: const Icon(Icons.person),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                      ),
                      validator: (value) {
                        if (value == null || value.isEmpty) {
                          return 'Please enter recipient address';
                        }
                        return null;
                      },
                      maxLines: 1,
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            
            // Token Selector
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Token',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 8,
                      children: _tokens.map((token) {
                        final isSelected = _selectedToken == token;
                        return ChoiceChip(
                          label: Text(token),
                          selected: isSelected,
                          onSelected: (selected) {
                            if (selected) {
                              setState(() => _selectedToken = token);
                            }
                          },
                        );
                      }).toList(),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            
            // Amount
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          'Amount',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        TextButton(
                          onPressed: () {
                            // Set max amount
                            _amountController.text = '1.0';
                          },
                          child: const Text('MAX'),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _amountController,
                      decoration: InputDecoration(
                        hintText: '0.00',
                        suffixText: _selectedToken,
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                      ),
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      validator: (value) {
                        if (value == null || value.isEmpty) {
                          return 'Please enter amount';
                        }
                        final amount = double.tryParse(value);
                        if (amount == null || amount <= 0) {
                          return 'Please enter valid amount';
                        }
                        return null;
                      },
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // Wallet password (or unlock_token) required by /auto-send and
            // /send to decrypt the seed server-side. The private key never
            // leaves the backend.
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Wallet password',
                    style: TextStyle(
                      color: AppColors.textSecondary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 8),
                  TextFormField(
                    controller: _passwordController,
                    obscureText: _obscurePassword,
                    decoration: InputDecoration(
                      hintText: 'Enter wallet password',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      suffixIcon: IconButton(
                        icon: Icon(
                          _obscurePassword
                              ? Icons.visibility_off
                              : Icons.visibility,
                        ),
                        onPressed: () => setState(
                            () => _obscurePassword = !_obscurePassword),
                      ),
                    ),
                    validator: (value) {
                      if (value == null || value.isEmpty) {
                        return 'Password is required to sign the transaction';
                      }
                      return null;
                    },
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            
            // Error Message
            if (_errorMessage != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.red.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: Colors.red.withOpacity(0.3)),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.error_outline, color: Colors.red),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        _errorMessage!,
                        style: const TextStyle(color: Colors.red),
                      ),
                    ),
                  ],
                ),
              ),
            
            const SizedBox(height: 24),
            
            // Send Button
            SizedBox(
              height: 56,
              child: ElevatedButton(
                onPressed: _isLoading ? null : _sendTransaction,
                style: ElevatedButton.styleFrom(
                  backgroundColor: const Color(0xFFFF6B35),
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
                child: _isLoading
                    ? const SizedBox(
                        height: 24,
                        width: 24,
                        child: CircularProgressIndicator(
                          color: Colors.white,
                          strokeWidth: 2,
                        ),
                      )
                    : Text(
                        'Send $_selectedToken',
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
