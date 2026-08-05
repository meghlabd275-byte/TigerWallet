import 'package:flutter/material.dart';
import '../models/admin_models.dart';
import '../services/admin_api_service.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  final AdminApiService _apiService = AdminApiService();
  AnalyticsData? _analyticsData;
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    loadDashboard();
  }

  Future<void> loadDashboard() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final data = await _apiService.getAnalyticsOverview();
      setState(() {
        _analyticsData = data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Dashboard'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadDashboard,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text('Error: $_error'))
              : _analyticsData != null
                  ? _buildDashboard()
                  : const Center(child: Text('No data')),
    );
  }

  Widget _buildDashboard() {
    final data = _analyticsData!;
    return RefreshIndicator(
      onRefresh: loadDashboard,
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            GridView.count(
              crossAxisCount: 2,
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              crossAxisSpacing: 16,
              mainAxisSpacing: 16,
              childAspectRatio: 1.5,
              children: [
                _buildStatCard(
                  'Total Users',
                  data.totalUsers.toString(),
                  Icons.people,
                  Colors.blue,
                ),
                _buildStatCard(
                  'Active Users',
                  data.activeUsers.toString(),
                  Icons.person,
                  Colors.green,
                ),
                _buildStatCard(
                  'Total Volume',
                  data.totalVolume,
                  Icons.attach_money,
                  Colors.orange,
                ),
                _buildStatCard(
                  'Daily Transactions',
                  data.dailyTransactions.toString(),
                  Icons.swap_horiz,
                  Colors.purple,
                ),
                _buildStatCard(
                  'Total Fees',
                  data.totalFees,
                  Icons.receipt,
                  Colors.red,
                ),
                _buildStatCard(
                  'Pending KYC',
                  data.pendingKyc.toString(),
                  Icons.verified_user,
                  Colors.amber,
                ),
              ],
            ),
            const SizedBox(height: 24),
            _buildSystemHealthCard(data.systemHealth),
          ],
        ),
      ),
    );
  }

  Widget _buildStatCard(String title, String value, IconData icon, Color color) {
    return Card(
      elevation: 2,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Icon(icon, color: color, size: 28),
              ],
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 12,
                    color: Colors.grey,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  value,
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSystemHealthCard(String health) {
    final isHealthy = health == 'healthy' || health == '99.9%';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Icon(
              isHealthy ? Icons.check_circle : Icons.warning,
              color: isHealthy ? Colors.green : Colors.orange,
              size: 32,
            ),
            const SizedBox(width: 16),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'System Health',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                Text(
                  health,
                  style: TextStyle(
                    color: isHealthy ? Colors.green : Colors.orange,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class UsersScreen extends StatefulWidget {
  const UsersScreen({super.key});

  @override
  State<UsersScreen> createState() => _UsersScreenState();
}

class _UsersScreenState extends State<UsersScreen> {
  final AdminApiService _apiService = AdminApiService();
  List<PlatformUser> _users = [];
  bool _isLoading = true;
  String? _error;
  String? _statusFilter;
  String? _kycFilter;
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    loadUsers();
  }

  Future<void> loadUsers() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final response = await _apiService.getUsers(
        status: _statusFilter,
        kycStatus: _kycFilter,
        search: _searchQuery.isEmpty ? null : _searchQuery,
      );
      setState(() {
        _users = response.data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Users'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadUsers,
          ),
        ],
      ),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator())
                : _error != null
                    ? Center(child: Text('Error: $_error'))
                    : _users.isEmpty
                        ? const Center(child: Text('No users found'))
                        : _buildUsersList(),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterBar() {
    return Container(
      padding: const EdgeInsets.all(16),
      color: Colors.grey[100],
      child: Row(
        children: [
          Expanded(
            child: TextField(
              decoration: const InputDecoration(
                hintText: 'Search users...',
                prefixIcon: Icon(Icons.search),
                border: OutlineInputBorder(),
                isDense: true,
              ),
              onChanged: (value) {
                _searchQuery = value;
                loadUsers();
              },
            ),
          ),
          const SizedBox(width: 16),
          DropdownButton<String>(
            value: _statusFilter,
            hint: const Text('Status'),
            items: const [
              DropdownMenuItem(value: null, child: Text('All')),
              DropdownMenuItem(value: 'active', child: Text('Active')),
              DropdownMenuItem(value: 'pending', child: Text('Pending')),
              DropdownMenuItem(value: 'suspended', child: Text('Suspended')),
              DropdownMenuItem(value: 'banned', child: Text('Banned')),
            ],
            onChanged: (value) {
              setState(() => _statusFilter = value);
              loadUsers();
            },
          ),
        ],
      ),
    );
  }

  Widget _buildUsersList() {
    return RefreshIndicator(
      onRefresh: loadUsers,
      child: ListView.builder(
        itemCount: _users.length,
        itemBuilder: (context, index) {
          final user = _users[index];
          return ListTile(
            leading: CircleAvatar(
              child: Text(user.email[0].toUpperCase()),
            ),
            title: Text(user.email),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('ID: ${user.id}'),
                Text('KYC: ${user.kycStatus.name} (L${user.kycLevel})'),
              ],
            ),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _buildStatusBadge(user.status),
                PopupMenuButton(
                  itemBuilder: (context) => [
                    if (user.status == UserStatus.active) ...[
                      const PopupMenuItem(
                        value: 'suspend',
                        child: Text('Suspend'),
                      ),
                      const PopupMenuItem(
                        value: 'ban',
                        child: Text('Ban'),
                      ),
                    ] else
                      const PopupMenuItem(
                        value: 'activate',
                        child: Text('Activate'),
                      ),
                  ],
                  onSelected: (value) async {
                    if (value == 'suspend') {
                      await _apiService.suspendUser(user.id, 'Suspended by admin');
                    } else if (value == 'ban') {
                      await _apiService.banUser(user.id, 'Banned by admin');
                    } else if (value == 'activate') {
                      await _apiService.activateUser(user.id);
                    }
                    loadUsers();
                  },
                ),
              ],
            ),
            isThreeLine: true,
          );
        },
      ),
    );
  }

  Widget _buildStatusBadge(UserStatus status) {
    Color color;
    switch (status) {
      case UserStatus.active:
        color = Colors.green;
        break;
      case UserStatus.pending:
        color = Colors.orange;
        break;
      case UserStatus.suspended:
        color = Colors.yellow;
        break;
      case UserStatus.banned:
        color = Colors.red;
        break;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.2),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        status.name.toUpperCase(),
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
}

class TransactionsScreen extends StatefulWidget {
  const TransactionsScreen({super.key});

  @override
  State<TransactionsScreen> createState() => _TransactionsScreenState();
}

class _TransactionsScreenState extends State<TransactionsScreen> {
  final AdminApiService _apiService = AdminApiService();
  List<Transaction> _transactions = [];
  bool _isLoading = true;
  String? _error;
  String? _statusFilter;
  bool _flaggedOnly = false;

  @override
  void initState() {
    super.initState();
    loadTransactions();
  }

  Future<void> loadTransactions() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final response = await _apiService.getTransactions(
        status: _statusFilter,
        flagged: _flaggedOnly ? true : null,
      );
      setState(() {
        _transactions = response.data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Transactions'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadTransactions,
          ),
        ],
      ),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator())
                : _error != null
                    ? Center(child: Text('Error: $_error'))
                    : _transactions.isEmpty
                        ? const Center(child: Text('No transactions'))
                        : _buildTransactionsList(),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterBar() {
    return Container(
      padding: const EdgeInsets.all(16),
      color: Colors.grey[100],
      child: Row(
        children: [
          DropdownButton<String>(
            value: _statusFilter,
            hint: const Text('Status'),
            items: const [
              DropdownMenuItem(value: null, child: Text('All')),
              DropdownMenuItem(value: 'pending', child: Text('Pending')),
              DropdownMenuItem(value: 'confirmed', child: Text('Confirmed')),
              DropdownMenuItem(value: 'failed', child: Text('Failed')),
            ],
            onChanged: (value) {
              setState(() => _statusFilter = value);
              loadTransactions();
            },
          ),
          const SizedBox(width: 16),
          FilterChip(
            label: const Text('Flagged Only'),
            selected: _flaggedOnly,
            onSelected: (value) {
              setState(() => _flaggedOnly = value);
              loadTransactions();
            },
          ),
        ],
      ),
    );
  }

  Widget _buildTransactionsList() {
    return RefreshIndicator(
      onRefresh: loadTransactions,
      child: ListView.builder(
        itemCount: _transactions.length,
        itemBuilder: (context, index) {
          final tx = _transactions[index];
          return ListTile(
            leading: Icon(
              _getTransactionIcon(tx.type),
              color: _getTransactionColor(tx.status),
            ),
            title: Text(tx.shortHash),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${tx.amount} ${tx.token}'),
                Text(tx.chain),
              ],
            ),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _buildTransactionStatusBadge(tx.status),
                if (tx.flagged)
                  const Padding(
                    padding: EdgeInsets.only(left: 8),
                    child: Icon(Icons.warning, color: Colors.red, size: 16),
                  ),
              ],
            ),
            isThreeLine: true,
          );
        },
      ),
    );
  }

  IconData _getTransactionIcon(TransactionType type) {
    switch (type) {
      case TransactionType.transfer:
        return Icons.swap_horiz;
      case TransactionType.swap:
        return Icons.swap_horizontal_circle;
      case TransactionType.stake:
        return Icons.lock;
      case TransactionType.unstake:
        return Icons.lock_open;
      case TransactionType.bridge:
        return Icons.link;
      case TransactionType.withdraw:
        return Icons.arrow_downward;
      case TransactionType.deposit:
        return Icons.arrow_upward;
      case TransactionType.mint:
        return Icons.add_circle;
      case TransactionType.burn:
        return Icons.remove_circle;
    }
  }

  Color _getTransactionColor(TransactionStatus status) {
    switch (status) {
      case TransactionStatus.confirmed:
        return Colors.green;
      case TransactionStatus.pending:
        return Colors.orange;
      case TransactionStatus.failed:
        return Colors.red;
    }
  }

  Widget _buildTransactionStatusBadge(TransactionStatus status) {
    Color color = _getTransactionColor(status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.2),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        status.name.toUpperCase(),
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
}

class KYCScreen extends StatefulWidget {
  const KYCScreen({super.key});

  @override
  State<KYCScreen> createState() => _KYCScreenState();
}

class _KYCScreenState extends State<KYCScreen> {
  final AdminApiService _apiService = AdminApiService();
  List<KYCApplication> _applications = [];
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    loadApplications();
  }

  Future<void> loadApplications() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final response = await _apiService.getKYCApplications();
      setState(() {
        _applications = response.data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('KYC Verification'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadApplications,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text('Error: $_error'))
              : _applications.isEmpty
                  ? const Center(child: Text('No KYC applications'))
                  : _buildList(),
    );
  }

  Widget _buildList() {
    return RefreshIndicator(
      onRefresh: loadApplications,
      child: ListView.builder(
        itemCount: _applications.length,
        itemBuilder: (context, index) {
          final app = _applications[index];
          return ListTile(
            leading: CircleAvatar(
              child: Text('L${app.level}'),
            ),
            title: Text(app.userEmail),
            subtitle: Text('Submitted: ${app.submittedAt}'),
            trailing: app.status == KYCApplicationStatus.pending
                ? Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.check, color: Colors.green),
                        onPressed: () async {
                          await _apiService.approveKYC(app.id);
                          loadApplications();
                        },
                      ),
                      IconButton(
                        icon: const Icon(Icons.close, color: Colors.red),
                        onPressed: () async {
                          await _apiService.rejectKYC(app.id, 'Rejected by admin');
                          loadApplications();
                        },
                      ),
                    ],
                  )
                : _buildKYCStatusBadge(app.status),
            isThreeLine: true,
          );
        },
      ),
    );
  }

  Widget _buildKYCStatusBadge(KYCApplicationStatus status) {
    Color color;
    switch (status) {
      case KYCApplicationStatus.approved:
        color = Colors.green;
        break;
      case KYCApplicationStatus.pending:
        color = Colors.orange;
        break;
      case KYCApplicationStatus.rejected:
        color = Colors.red;
        break;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.2),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        status.name.toUpperCase(),
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
}

class TokensScreen extends StatefulWidget {
  const TokensScreen({super.key});

  @override
  State<TokensScreen> createState() => _TokensScreenState();
}

class _TokensScreenState extends State<TokensScreen> {
  final AdminApiService _apiService = AdminApiService();
  List<Token> _tokens = [];
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    loadTokens();
  }

  Future<void> loadTokens() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final response = await _apiService.getTokens();
      setState(() {
        _tokens = response.data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Tokens'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadTokens,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text('Error: $_error'))
              : _tokens.isEmpty
                  ? const Center(child: Text('No tokens'))
                  : _buildList(),
    );
  }

  Widget _buildList() {
    return RefreshIndicator(
      onRefresh: loadTokens,
      child: ListView.builder(
        itemCount: _tokens.length,
        itemBuilder: (context, index) {
          final token = _tokens[index];
          return ListTile(
            leading: token.logoUrl != null
                ? Image.network(token.logoUrl!, width: 40, height: 40)
                : CircleAvatar(child: Text(token.symbol[0])),
            title: Row(
              children: [
                Text(token.name),
                if (token.isVerified)
                  const Padding(
                    padding: EdgeInsets.only(left: 4),
                    child: Icon(Icons.verified, color: Colors.blue, size: 16),
                  ),
              ],
            ),
            subtitle: Text(token.symbol),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (token.price != null) Text('\$${token.price}'),
                const SizedBox(width: 8),
                token.isActive
                    ? IconButton(
                        icon: const Icon(Icons.pause_circle),
                        onPressed: () async {
                          await _apiService.deactivateToken(token.id);
                          loadTokens();
                        },
                      )
                    : IconButton(
                        icon: const Icon(Icons.play_circle),
                        onPressed: () async {
                          await _apiService.activateToken(token.id);
                          loadTokens();
                        },
                      ),
              ],
            ),
            isThreeLine: token.price != null,
          );
        },
      ),
    );
  }
}

class WithdrawalsScreen extends StatefulWidget {
  const WithdrawalsScreen({super.key});

  @override
  State<WithdrawalsScreen> createState() => _WithdrawalsScreenState();
}

class _WithdrawalsScreenState extends State<WithdrawalsScreen> {
  final AdminApiService _apiService = AdminApiService();
  List<WithdrawalRequest> _withdrawals = [];
  bool _isLoading = true;
  String? _error;
  String? _statusFilter;

  @override
  void initState() {
    super.initState();
    loadWithdrawals();
  }

  Future<void> loadWithdrawals() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final response = await _apiService.getWithdrawals(status: _statusFilter);
      setState(() {
        _withdrawals = response.data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Withdrawals'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadWithdrawals,
          ),
        ],
      ),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator())
                : _error != null
                    ? Center(child: Text('Error: $_error'))
                    : _withdrawals.isEmpty
                        ? const Center(child: Text('No withdrawals'))
                        : _buildList(),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterBar() {
    return Container(
      padding: const EdgeInsets.all(16),
      color: Colors.grey[100],
      child: DropdownButton<String>(
        value: _statusFilter,
        hint: const Text('Status'),
        items: const [
          DropdownMenuItem(value: null, child: Text('All')),
          DropdownMenuItem(value: 'pending', child: Text('Pending')),
          DropdownMenuItem(value: 'approved', child: Text('Approved')),
          DropdownMenuItem(value: 'completed', child: Text('Completed')),
          DropdownMenuItem(value: 'rejected', child: Text('Rejected')),
        ],
        onChanged: (value) {
          setState(() => _statusFilter = value);
          loadWithdrawals();
        },
      ),
    );
  }

  Widget _buildList() {
    return RefreshIndicator(
      onRefresh: loadWithdrawals,
      child: ListView.builder(
        itemCount: _withdrawals.length,
        itemBuilder: (context, index) {
          final withdrawal = _withdrawals[index];
          return ListTile(
            leading: const Icon(Icons.arrow_downward),
            title: Text('${withdrawal.amount} ${withdrawal.token}'),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(withdrawal.userEmail),
                Text('To: ${withdrawal.toAddress.substring(0, 10)}...'),
              ],
            ),
            trailing: withdrawal.status == WithdrawalStatus.pending
                ? Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.check, color: Colors.green),
                        onPressed: () async {
                          await _apiService.approveWithdrawal(withdrawal.id);
                          loadWithdrawals();
                        },
                      ),
                      IconButton(
                        icon: const Icon(Icons.close, color: Colors.red),
                        onPressed: () async {
                          await _apiService.rejectWithdrawal(
                              withdrawal.id, 'Rejected by admin');
                          loadWithdrawals();
                        },
                      ),
                    ],
                  )
                : _buildWithdrawalStatusBadge(withdrawal.status),
            isThreeLine: true,
          );
        },
      ),
    );
  }

  Widget _buildWithdrawalStatusBadge(WithdrawalStatus status) {
    Color color;
    switch (status) {
      case WithdrawalStatus.completed:
        color = Colors.green;
        break;
      case WithdrawalStatus.approved:
      case WithdrawalStatus.processing:
        color = Colors.blue;
        break;
      case WithdrawalStatus.pending:
        color = Colors.orange;
        break;
      case WithdrawalStatus.rejected:
      case WithdrawalStatus.failed:
        color = Colors.red;
        break;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.2),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        status.name.toUpperCase(),
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
}

class SystemScreen extends StatefulWidget {
  const SystemScreen({super.key});

  @override
  State<SystemScreen> createState() => _SystemScreenState();
}

class _SystemScreenState extends State<SystemScreen> {
  final AdminApiService _apiService = AdminApiService();
  Map<String, List<SystemStatus>>? _status;
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    loadStatus();
  }

  Future<void> loadStatus() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final status = await _apiService.getSystemStatus();
      setState(() {
        _status = status;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('System Status'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: loadStatus,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text('Error: $_error'))
              : _status != null
                  ? _buildStatus()
                  : const Center(child: Text('No data')),
    );
  }

  Widget _buildStatus() {
    return RefreshIndicator(
      onRefresh: loadStatus,
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildSection('Services', _status!['services'] ?? []),
            _buildSection('Databases', _status!['databases'] ?? []),
            _buildSection('Networks', _status!['networks'] ?? []),
          ],
        ),
      ),
    );
  }

  Widget _buildSection(String title, List<SystemStatus> items) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: const TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 8),
        ...items.map((item) => Card(
              child: ListTile(
                leading: Icon(
                  item.isHealthy ? Icons.check_circle : Icons.warning,
                  color: item.isHealthy ? Colors.green : Colors.red,
                ),
                title: Text(item.serviceName),
                subtitle: Text('Uptime: ${item.uptime} | Latency: ${item.latency}'),
                trailing: Text(item.status),
              ),
            )),
        const SizedBox(height: 16),
      ],
    );
  }
}
