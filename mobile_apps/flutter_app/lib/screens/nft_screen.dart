import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// NFT Screen - View and manage NFT collections
class NFTScreen extends StatefulWidget {
  const NFTScreen({super.key});

  @override
  State<NFTScreen> createState() => _NFTScreenState();
}

class _NFTScreenState extends State<NFTScreen> {
  List<Map<String, dynamic>> _nfts = [];
  bool _isLoading = true;
  String _selectedFilter = 'all';

  @override
  void initState() {
    super.initState();
    _loadNFTs();
  }

  Future<void> _loadNFTs() async {
    setState(() => _isLoading = true);
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/nfts'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _nfts = List<Map<String, dynamic>>.from(data['nfts'] ?? []);
        });
      }
    } catch (e) {
      _loadDemoNFTs();
    }
    setState(() => _isLoading = false);
  }

  void _loadDemoNFTs() {
    setState(() {
      _nfts = [
        {
          'id': '1',
          'name': 'Bored Ape #8823',
          'collection': 'Bored Ape Yacht Club',
          'image': 'https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ',
          'chain': 'Ethereum',
          'floor_price': '35.5',
          'last_sale': '42.0',
        },
        {
          'id': '2',
          'name': 'CryptoPunk #3456',
          'collection': 'CryptoPunks',
          'image': 'https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ',
          'chain': 'Ethereum',
          'floor_price': '65.0',
          'last_sale': '78.5',
        },
        {
          'id': '3',
          'name': 'DeGod #420',
          'collection': 'DeGods',
          'image': 'https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ',
          'chain': 'Solana',
          'floor_price': '250',
          'last_sale': '280',
        },
        {
          'id': '4',
          'name': 'Mad Lads #1234',
          'collection': 'Mad Lads',
          'image': 'https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ',
          'chain': 'Solana',
          'floor_price': '180',
          'last_sale': '195',
        },
        {
          'id': '5',
          'name': 'Azuki #7890',
          'collection': 'Azuki',
          'image': 'https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ',
          'chain': 'Ethereum',
          'floor_price': '12.5',
          'last_sale': '15.2',
        },
        {
          'id': '6',
          'name': 'StepN Shoe #5678',
          'collection': 'StepN',
          'image': 'https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ',
          'chain': 'Solana',
          'floor_price': '8.5',
          'last_sale': '10.2',
        },
      ];
    });
  }

  List<Map<String, dynamic>> get _filteredNFTs {
    if (_selectedFilter == 'all') return _nfts;
    return _nfts.where((nft) => nft['chain'] == _selectedFilter).toList();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('NFTs', style: TextStyle(color: AppColors.textPrimary)),
        actions: [
          IconButton(
            icon: const Icon(Icons.search, color: AppColors.textPrimary),
            onPressed: () {},
          ),
          IconButton(
            icon: const Icon(Icons.grid_view, color: AppColors.textPrimary),
            onPressed: () {},
          ),
        ],
      ),
      body: Column(
        children: [
          _buildStats(),
          _buildFilters(),
          Expanded(child: _buildNFTGrid()),
        ],
      ),
    );
  }

  Widget _buildStats() {
    double totalValue = 0;
    for (var nft in _nfts) {
      totalValue += (double.tryParse(nft['floor_price'] ?? '0') ?? 0);
    }

    return Container(
      margin: const EdgeInsets.all(16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [AppColors.primary, AppColors.primaryDark],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _buildStatItem('Total NFTs', '${_nfts.length}'),
          _buildStatItem('Collections', '${_nfts.length}'),
          _buildStatItem('Value', '\$${totalValue.toStringAsFixed(1)}K'),
        ],
      ),
    );
  }

  Widget _buildStatItem(String label, String value) {
    return Column(
      children: [
        Text(value,
            style: const TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.bold,
                color: Colors.white)),
        const SizedBox(height: 4),
        Text(label,
            style: const TextStyle(color: Colors.white70, fontSize: 12)),
      ],
    );
  }

  Widget _buildFilters() {
    final filters = ['all', 'Ethereum', 'Solana', 'Polygon', 'Avalanche'];
    
    return Container(
      height: 50,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: ListView.builder(
        scrollDirection: Axis.horizontal,
        itemCount: filters.length,
        itemBuilder: (context, index) {
          final filter = filters[index];
          final isSelected = _selectedFilter == filter;
          
          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: FilterChip(
              label: Text(filter == 'all' ? 'All' : filter),
              selected: isSelected,
              onSelected: (selected) {
                setState(() => _selectedFilter = filter);
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

  Widget _buildNFTGrid() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_filteredNFTs.isEmpty) {
      return const Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.image_outlined, size: 64, color: AppColors.textSecondary),
            SizedBox(height: 16),
            Text('No NFTs found', style: TextStyle(color: AppColors.textSecondary)),
          ],
        ),
      );
    }

    return GridView.builder(
      padding: const EdgeInsets.all(16),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        childAspectRatio: 0.75,
        crossAxisSpacing: 12,
        mainAxisSpacing: 12,
      ),
      itemCount: _filteredNFTs.length,
      itemBuilder: (context, index) {
        final nft = _filteredNFTs[index];
        return _buildNFTCard(nft);
      },
    );
  }

  Widget _buildNFTCard(Map<String, dynamic> nft) {
    return GestureDetector(
      onTap: () => _showNFCDetails(nft),
      child: Container(
        decoration: BoxDecoration(
          color: AppColors.cardBackground,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: AppColors.border.withOpacity(0.1)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // NFT Image
            Expanded(
              flex: 3,
              child: Container(
                decoration: BoxDecoration(
                  color: AppColors.primary.withOpacity(0.1),
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(16),
                  ),
                ),
                child: const Center(
                  child: Icon(Icons.image, size: 48, color: AppColors.primary),
                ),
              ),
            ),
            // NFT Info
            Expanded(
              flex: 2,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      nft['name'] ?? '',
                      style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        fontSize: 13,
                        color: AppColors.textPrimary,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      nft['collection'] ?? '',
                      style: TextStyle(
                        color: AppColors.textSecondary,
                        fontSize: 11,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const Spacer(),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Row(
                          children: [
                            const Icon(Icons.show_chart,
                                size: 14, color: AppColors.success),
                            const SizedBox(width: 2),
                            Text(
                              '${nft['floor_price']} ETH',
                              style: const TextStyle(
                                fontWeight: FontWeight.bold,
                                fontSize: 12,
                                color: AppColors.success,
                              ),
                            ),
                          ],
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: AppColors.primary.withOpacity(0.1),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            nft['chain']?.substring(0, 2).toUpperCase() ?? '',
                            style: const TextStyle(
                              fontSize: 9,
                              color: AppColors.primary,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ),
                      ],
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

  void _showNFCDetails(Map<String, dynamic> nft) {
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => DraggableScrollableSheet(
        initialChildSize: 0.7,
        minChildSize: 0.5,
        maxChildSize: 0.9,
        expand: false,
        builder: (context, scrollController) => SingleChildScrollView(
          controller: scrollController,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Handle
              Center(
                child: Container(
                  margin: const EdgeInsets.only(top: 12),
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: AppColors.border,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              // NFT Image
              Container(
                margin: const EdgeInsets.all(20),
                height: 300,
                decoration: BoxDecoration(
                  color: AppColors.primary.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: const Center(
                  child: Icon(Icons.image, size: 100, color: AppColors.primary),
                ),
              ),
              // NFT Info
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      nft['name'] ?? '',
                      style: const TextStyle(
                        fontSize: 24,
                        fontWeight: FontWeight.bold,
                        color: AppColors.textPrimary,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      nft['collection'] ?? '',
                      style: TextStyle(
                        fontSize: 16,
                        color: AppColors.textSecondary,
                      ),
                    ),
                    const SizedBox(height: 20),
                    // Stats
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceAround,
                      children: [
                        _buildDetailStat('Floor Price', '${nft['floor_price']} ETH'),
                        _buildDetailStat('Last Sale', '${nft['last_sale']} ETH'),
                        _buildDetailStat('Chain', nft['chain'] ?? ''),
                      ],
                    ),
                    const SizedBox(height: 24),
                    // Actions
                    Row(
                      children: [
                        Expanded(
                          child: OutlinedButton(
                            onPressed: () {},
                            style: OutlinedButton.styleFrom(
                              padding: const EdgeInsets.symmetric(vertical: 14),
                            ),
                            child: const Text('Send'),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: ElevatedButton(
                            onPressed: () {},
                            style: ElevatedButton.styleFrom(
                              backgroundColor: AppColors.primary,
                              padding: const EdgeInsets.symmetric(vertical: 14),
                            ),
                            child: const Text('View on Explorer'),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 20),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildDetailStat(String label, String value) {
    return Column(
      children: [
        Text(label, style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
        const SizedBox(height: 4),
        Text(value,
            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
      ],
    );
  }
}
