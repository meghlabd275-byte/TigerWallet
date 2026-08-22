/**
 * TigerWallet Admin - More Screen
 * Navigation hub for all admin governance surfaces. Theme-aware.
 */

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class MoreScreen extends StatelessWidget {
  const MoreScreen({Key? key}) : super(key: key);

  static const List<_MoreItem> _items = [
    _MoreItem(Icons.payments, 'Crypto Cards', '/crypto-cards'),
    _MoreItem(Icons.show_chart, 'Margin Trading', '/margin-trading'),
    _MoreItem(Icons.flag, 'Feature Flags', '/feature-flags'),
    _MoreItem(Icons.receipt_long, 'Billing', '/billing'),
    _MoreItem(Icons.store, 'P2P Merchants', '/p2p-merchants'),
    _MoreItem(Icons.water, 'Liquidity', '/liquidity'),
    _MoreItem(Icons.account_balance_wallet, 'Master Wallet', '/master-wallet'),
    _MoreItem(Icons.trending_up, 'Futures', '/domain/futures'),
    _MoreItem(Icons.tune, 'Options', '/domain/options'),
    _MoreItem(Icons.copy, 'Copy Trading', '/domain/copy-trading'),
    _MoreItem(Icons.swap_vert, 'Convert', '/domain/convert'),
    _MoreItem(Icons.login, 'On-Ramp', '/domain/onramp'),
    _MoreItem(Icons.logout, 'Off-Ramp', '/domain/offramp'),
    _MoreItem(Icons.people_alt, 'P2P Clients', '/domain/p2p-clients'),
    _MoreItem(Icons.handshake, 'Partners', '/domain/partners'),
    _MoreItem(Icons.card_giftcard, 'Rewards', '/domain/rewards'),
    _MoreItem(Icons.campaign, 'Marketing', '/domain/marketing'),
    _MoreItem(Icons.smart_toy, 'Bots', '/domain/bots'),
    _MoreItem(Icons.groups, 'Bots Clients', '/domain/bots-clients'),
    _MoreItem(Icons.engineering, 'Project Teams', '/domain/project-teams'),
    _MoreItem(Icons.water_drop, 'Liquidity Sources', '/domain/liquidity-sources'),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('More')),
      body: ListView.builder(
        itemCount: _items.length,
        itemBuilder: (ctx, i) {
          final item = _items[i];
          return ListTile(
            leading: Icon(item.icon),
            title: Text(item.title),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.go(item.route),
          );
        },
      ),
    );
  }
}

class _MoreItem {
  final IconData icon;
  final String title;
  final String route;
  const _MoreItem(this.icon, this.title, this.route);
}
