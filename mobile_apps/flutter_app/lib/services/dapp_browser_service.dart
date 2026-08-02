import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// DApp Browser Service for Flutter App
class DAppBrowserService {
  static final DAppBrowserService _instance = DAppBrowserService._internal();
  factory DAppBrowserService() => _instance;
  DAppBrowserService._internal();

  final String _baseUrl = 'https://api.tigerwallet.com/v1/dapps';

  /// Get list of DApps
  Future<List<DApp>> getDApps({String? category}) async {
    // In production, fetch from API
    // For now, return popular DApps
    return _popularDApps;
  }

  /// Search DApps
  Future<List<DApp>> searchDApps(String query) async {
    final lowercaseQuery = query.toLowerCase();
    return _popularDApps.where((dapp) =>
      dapp.name.toLowerCase().contains(lowercaseQuery) ||
      (dapp.description?.toLowerCase().contains(lowercaseQuery) ?? false)
    ).toList();
  }

  /// Get DApp categories
  Future<List<String>> getCategories() async {
    return [
      'DeFi',
      'NFT',
      'Games',
      'Social',
      'Tools',
      'Bridge',
      'Staking',
      'Exchange',
    ];
  }

  /// Get featured DApps
  Future<List<DApp>> getFeaturedDApps() async {
    return _popularDApps.where((dapp) => dapp.isFeatured).toList();
  }

  /// Get popular DApps
  Future<List<DApp>> getPopularDApps() async {
    return _popularDApps;
  }

  // Popular DApps list
  static final List<DApp> _popularDApps = [
    DApp(
      id: '1',
      name: 'Uniswap',
      description: 'Decentralized trading protocol',
      url: 'https://app.uniswap.org',
      logoUrl: 'https://cryptologos.cc/logos/uniswap-uni-logo.png',
      category: 'DeFi',
      chains: ['ethereum', 'arbitrum', 'optimism'],
      isFeatured: true,
    ),
    DApp(
      id: '2',
      name: 'Aave',
      description: 'Non-custodial liquidity protocol',
      url: 'https://app.aave.com',
      logoUrl: 'https://cryptologos.cc/logos/aave-aave-logo.png',
      category: 'DeFi',
      chains: ['ethereum', 'polygon', 'avalanche'],
      isFeatured: true,
    ),
    DApp(
      id: '3',
      name: 'OpenSea',
      description: 'NFT marketplace',
      url: 'https://opensea.io',
      logoUrl: 'https://cryptologos.cc/logos/opensea-os-logo.png',
      category: 'NFT',
      chains: ['ethereum', 'polygon'],
      isFeatured: true,
    ),
    DApp(
      id: '4',
      name: 'Compound',
      description: 'Algorithmic money market protocol',
      url: 'https://app.compound.finance',
      logoUrl: 'https://cryptologos.cc/logos/compound-comp-logo.png',
      category: 'DeFi',
      chains: ['ethereum'],
    ),
    DApp(
      id: '5',
      name: 'Curve',
      description: 'Stable asset exchange',
      url: 'https://curve.fi',
      logoUrl: 'https://cryptologos.cc/logos/curve-dao-token-crv-logo.png',
      category: 'DeFi',
      chains: ['ethereum', 'polygon', 'avalanche'],
    ),
    DApp(
      id: '6',
      name: 'LooksRare',
      description: 'NFT marketplace with rewards',
      url: 'https://looksrare.org',
      logoUrl: 'https://cryptologos.cc/logos/looksrare-looks-logo.png',
      category: 'NFT',
      chains: ['ethereum'],
    ),
    DApp(
      id: '7',
      name: '1inch',
      description: 'DEX aggregator',
      url: 'https://app.1inch.io',
      logoUrl: 'https://cryptologos.cc/logos/1inch-1inch-logo.png',
      category: 'DeFi',
      chains: ['ethereum', 'polygon', 'bsc'],
    ),
    DApp(
      id: '8',
      name: 'Sushiswap',
      description: 'Decentralized exchange',
      url: 'https://sushi.com',
      logoUrl: 'https://cryptologos.cc/logos/sushi-sushi-logo.png',
      category: 'DeFi',
      chains: ['ethereum', 'polygon', 'avalanche'],
    ),
    DApp(
      id: '9',
      name: 'PancakeSwap',
      description: 'DEX on BNB Chain',
      url: 'https://pancakeswap.finance',
      logoUrl: 'https://cryptologos.cc/logos/pancakeswap-cake-logo.png',
      category: 'DeFi',
      chains: ['bsc', 'ethereum'],
    ),
    DApp(
      id: '10',
      name: 'Blur',
      description: 'NFT marketplace for traders',
      url: 'https://blur.io',
      logoUrl: 'https://cryptologos.cc/logos/blur-blur-logo.png',
      category: 'NFT',
      chains: ['ethereum'],
    ),
  ];
}

/// DApp model
class DApp {
  final String id;
  final String name;
  final String? description;
  final String url;
  final String? logoUrl;
  final String category;
  final List<String> chains;
  final bool isFeatured;

  DApp({
    required this.id,
    required this.name,
    this.description,
    required this.url,
    this.logoUrl,
    required this.category,
    required this.chains,
    this.isFeatured = false,
  });
}

/// DApp Browser Widget
class DAppBrowser extends StatefulWidget {
  final String url;
  final String? title;
  final Function(String)? onUrlChanged;
  final Function(String)? onPageFinished;

  const DAppBrowser({
    Key? key,
    required this.url,
    this.title,
    this.onUrlChanged,
    this.onPageFinished,
  }) : super(key: key);

  @override
  State<DAppBrowser> createState() => _DAppBrowserState();
}

class _DAppBrowserState extends State<DAppBrowser> {
  late WebViewController _controller;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _initWebView();
  }

  void _initWebView() {
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (String url) {
            setState(() => _isLoading = true);
          },
          onPageFinished: (String url) {
            setState(() => _isLoading = false);
            widget.onPageFinished?.call(url);
          },
          onNavigationRequest: (NavigationRequest request) {
            // Handle navigation
            widget.onUrlChanged?.call(request.url);
            return NavigationDecision.navigate;
          },
        ),
      )
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        WebViewWidget(controller: _controller),
        if (_isLoading)
          const Center(
            child: CircularProgressIndicator(),
          ),
      ],
    );
  }
}
