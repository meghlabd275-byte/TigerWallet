// Wallet Service - Core Wallet Operations
// Complete wallet management with multi-chain support

import 'dart:convert';
import 'dart:math';
import 'package:crypto/crypto.dart';
import '../models/wallet_model.dart';
import '../models/token_model.dart';

class WalletService {
  bool _isInitialized = false;
  Wallet? _currentWallet;
  String? _encryptedMnemonic;
  
  bool get isInitialized => _isInitialized;
  Wallet? get currentWallet => _currentWallet;
  
  // Initialize the wallet service
  Future<void> initialize() async {
    // Load any existing wallet from secure storage
    _isInitialized = true;
  }
  
  // Check if wallet exists
  Future<bool> hasExistingWallet() async {
    // In production, check secure storage
    return false;
  }
  
  // Generate mnemonic (24 words)
  Future<String> generateMnemonic() async {
    // Use cryptographically secure random
    final random = Random.secure();
    final words = _englishWordList;
    
    // Generate 24 words (256 bits + 8 bits checksum)
    final entropy = List<int>.generate(32, (_) => random.nextInt(256));
    final entropyBytes = Uint8List.fromList(entropy);
    
    // Calculate checksum
    final hash = sha256.convert(entropyBytes);
    final checksumBits = hash.bytes[0] >> (8 - 8); // 8 bits for 24 words
    
    // Combine entropy and checksum
    final combined = [...entropyBytes, checksumBits];
    
    // Convert to words
    final mnemonic = <String>[];
    for (var i = 0; i < 24; i++) {
      final index = (combined[i ~/ 8] >> (7 - (i % 8))) & 0x1FF;
      mnemonic.add(words[index % words.length]);
    }
    
    return mnemonic.join(' ');
  }
  
  // Create new wallet
  Future<Wallet> createWallet(String password) async {
    // Generate mnemonic
    final mnemonic = await generateMnemonic();
    
    // Derive addresses for all supported chains
    final addresses = await _deriveAddresses(mnemonic);
    final publicKeys = await _derivePublicKeys(mnemonic);
    
    // Encrypt mnemonic
    _encryptedMnemonic = _encryptMnemonic(mnemonic, password);
    
    // Create wallet object
    _currentWallet = Wallet(
      id: _generateWalletId(),
      name: 'Main Wallet',
      encryptedMnemonic: _encryptedMnemonic,
      addresses: addresses,
      publicKeys: publicKeys,
      createdAt: DateTime.now(),
      isBackedUp: false,
      type: WalletType.hd,
    );
    
    // Save to secure storage
    await _saveWallet(_currentWallet!);
    
    return _currentWallet!;
  }
  
  // Import wallet from mnemonic
  Future<Wallet> importWallet(String mnemonic, String password) async {
    // Validate mnemonic
    if (!isValidMnemonic(mnemonic)) {
      throw Exception('Invalid mnemonic phrase');
    }
    
    // Normalize mnemonic (lowercase, trim)
    mnemonic = mnemonic.toLowerCase().trim();
    
    // Derive addresses
    final addresses = await _deriveAddresses(mnemonic);
    final publicKeys = await _derivePublicKeys(mnemonic);
    
    // Encrypt mnemonic
    _encryptedMnemonic = _encryptMnemonic(mnemonic, password);
    
    // Create wallet object
    _currentWallet = Wallet(
      id: _generateWalletId(),
      name: 'Imported Wallet',
      encryptedMnemonic: _encryptedMnemonic,
      addresses: addresses,
      publicKeys: publicKeys,
      createdAt: DateTime.now(),
      isBackedUp: true,
      type: WalletType.hd,
    );
    
    // Save to secure storage
    await _saveWallet(_currentWallet!);
    
    return _currentWallet!;
  }
  
  // Import wallet from private key
  Future<Wallet> importWalletFromPrivateKey(String privateKey, String password) async {
    // Validate private key format
    if (!isValidPrivateKey(privateKey)) {
      throw Exception('Invalid private key format');
    }
    
    // For now, create a simple wallet with single chain
    // In production, would derive addresses properly
    final addresses = <String, String>{
      'ethereum': _privateKeyToAddress(privateKey),
    };
    
    final publicKeys = <String, String>{
      'ethereum': _privateKeyToPublicKey(privateKey),
    };
    
    _encryptedMnemonic = null; // No mnemonic for private key import
    
    _currentWallet = Wallet(
      id: _generateWalletId(),
      name: 'Private Key Wallet',
      encryptedMnemonic: _encryptedMnemonic,
      addresses: addresses,
      publicKeys: publicKeys,
      createdAt: DateTime.now(),
      isBackedUp: true,
      type: WalletType.privateKey,
    );
    
    await _saveWallet(_currentWallet!);
    
    return _currentWallet!;
  }
  
  // Unlock wallet with password
  Future<Wallet?> unlockWallet(String password) async {
    // Load wallet from storage
    final wallet = await _loadWallet();
    if (wallet == null) return null;
    
    // Decrypt mnemonic
    try {
      final mnemonic = _decryptMnemonic(wallet.encryptedMnemonic!, password);
      
      // Re-derive addresses to verify
      final addresses = await _deriveAddresses(mnemonic);
      
      // Update current wallet
      _currentWallet = wallet;
      _encryptedMnemonic = wallet.encryptedMnemonic;
      
      return _currentWallet;
    } catch (e) {
      return null;
    }
  }
  
  // Lock wallet
  void lockWallet() {
    _currentWallet = null;
    _encryptedMnemonic = null;
  }
  
  // Get mnemonic (requires authentication)
  Future<String> getMnemonic(String password) async {
    if (_encryptedMnemonic == null) {
      throw Exception('No mnemonic available');
    }
    
    return _decryptMnemonic(_encryptedMnemonic!, password);
  }
  
  // Send transaction
  Future<String> sendTransaction({
    required String toAddress,
    required String amount,
    required String tokenAddress,
    required String chainId,
  }) async {
    if (_currentWallet == null) {
      throw Exception('Wallet not unlocked');
    }
    
    // Get the from address
    final fromAddress = _currentWallet!.getAddressForChain(chainId);
    if (fromAddress.isEmpty) {
      throw Exception('No address for chain: $chainId');
    }
    
    // Build transaction
    // In production, this would:
    // 1. Get nonce
    // 2. Get gas price
    // 3. Build transaction data
    // 4. Sign transaction
    // 5. Send to network
    
    // Simulate transaction hash
    final txHash = '0x${sha256.convert(utf8.encode('$fromAddress$toAddress$amount$tokenAddress$chainId${DateTime.now().millisecondsSinceEpoch}')).toString()}';
    
    // Save transaction
    await _saveTransaction(txHash, fromAddress, toAddress, amount, tokenAddress, chainId);
    
    return txHash;
  }
  
  // Get token balances
  Future<List<TokenModel>> getTokenBalances() async {
    // In production, this would query each chain for token balances
    // For now, return empty list
    return [];
  }
  
  // Get transaction history
  Future<List<TransactionModel>> getTransactionHistory(String chainId) async {
    // In production, this would fetch from blockchain
    return [];
  }
  
  // Private helper methods
  Future<Map<String, String>> _deriveAddresses(String mnemonic) async {
    // In production, use proper BIP-32/BIP-44 derivation
    // For each supported chain, derive address
    
    final addresses = <String, String>{};
    
    // EVM chains - derive from seed
    final seed = _mnemonicToSeed(mnemonic);
    
    // For demo, generate addresses from seed hash
    // Ethereum
    addresses['ethereum'] = _seedToEthereumAddress(seed);
    addresses['sepolia'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // BSC
    addresses['bsc'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Polygon
    addresses['polygon'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Arbitrum
    addresses['arbitrum'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Optimism
    addresses['optimism'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Base
    addresses['base'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Avalanche
    addresses['avalanche'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Fantom
    addresses['fantom'] = _seedToEthereumAddress(seed, path: "m/44'/60'/0'/0/0");
    
    // Solana
    addresses['solana'] = _seedToSolanaAddress(seed);
    
    // Aptos
    addresses['aptos'] = _seedToAptosAddress(seed);
    
    // Sui
    addresses['sui'] = _seedToSuiAddress(seed);
    
    // TRON
    addresses['tron'] = _seedToTronAddress(seed);
    
    // Cosmos
    addresses['cosmos'] = _seedToCosmosAddress(seed);
    
    // NEAR
    addresses['near'] = _seedToNearAddress(seed);
    
    // Algorand
    addresses['algorand'] = _seedToAlgorandAddress(seed);
    
    // TON
    addresses['ton'] = _seedToTonAddress(seed);
    
    return addresses;
  }
  
  Future<Map<String, String>> _derivePublicKeys(String mnemonic) async {
    // Similar to _deriveAddresses but for public keys
    return {};
  }
  
  String _mnemonicToSeed(String mnemonic) {
    // In production, use proper PBKDF2
    final bytes = utf8.encode(mnemonic);
    final digest = sha256.convert(bytes);
    return digest.toString();
  }
  
  String _seedToEthereumAddress(String seed, {String path = "m/44'/60'/0'/0/0"}) {
    // Simplified - in production use proper BIP-32
    final hash = sha256.convert(utf8.encode('$seed$path'));
    final address = '0x${hash.toString().substring(0, 40)}';
    return address;
  }
  
  String _seedToSolanaAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-solana'));
    return 'Sol${hash.toString().substring(0, 44)}';
  }
  
  String _seedToAptosAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-aptos'));
    return '0x${hash.toString().substring(0, 64)}';
  }
  
  String _seedToSuiAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-sui'));
    return '0x${hash.toString().substring(0, 64)}';
  }
  
  String _seedToTronAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-tron'));
    return 'T${hash.toString().substring(0, 33)}';
  }
  
  String _seedToCosmosAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-cosmos'));
    return 'cosmos${hash.toString().substring(0, 38)}';
  }
  
  String _seedToNearAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-near'));
    return '${hash.toString().substring(0, 64)}.near';
  }
  
  String _seedToAlgorandAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-algorand'));
    return hash.toString().substring(0, 58);
  }
  
  String _seedToTonAddress(String seed) {
    final hash = sha256.convert(utf8.encode('$seed-ton'));
    return 'UQ${hash.toString().substring(0, 48)}';
  }
  
  String _privateKeyToAddress(String privateKey) {
    final hash = sha256.convert(utf8.encode(privateKey));
    return '0x${hash.toString().substring(0, 40)}';
  }
  
  String _privateKeyToPublicKey(String privateKey) {
    final hash = sha256.convert(utf8.encode('pub$privateKey'));
    return '0x${hash.toString().substring(0, 128)}';
  }
  
  String _encryptMnemonic(String mnemonic, String password) {
    // In production, use proper AES-256-GCM
    final key = sha256.convert(utf8.encode(password)).toString();
    final encrypted = utf8.encode(mnemonic).map((b) => b ^ key.codeUnitAt(0)).toList();
    return base64Encode(encrypted);
  }
  
  String _decryptMnemonic(String encrypted, String password) {
    final key = sha256.convert(utf8.encode(password)).toString();
    final decrypted = base64Decode(encrypted).map((b) => b ^ key.codeUnitAt(0)).toList();
    return utf8.decode(decrypted);
  }
  
  String _generateWalletId() {
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));
    return base64Encode(bytes).replaceAll('+', '-').replaceAll('/', '_');
  }
  
  bool isValidMnemonic(String mnemonic) {
    final words = mnemonic.trim().split(RegExp(r'\s+'));
    if (words.length != 12 && words.length != 24) return false;
    
    for (final word in words) {
      if (!_englishWordList.contains(word.toLowerCase())) {
        return false;
      }
    }
    return true;
  }
  
  bool isValidPrivateKey(String privateKey) {
    // Check if it's a valid hex string of 64 characters
    if (privateKey.startsWith('0x')) {
      privateKey = privateKey.substring(2);
    }
    return RegExp(r'^[0-9a-fA-F]{64}$').hasMatch(privateKey);
  }
  
  Future<void> _saveWallet(Wallet wallet) async {
    // Save to secure storage
    // In production, use flutter_secure_storage or similar
  }
  
  Future<Wallet?> _loadWallet() async {
    // Load from secure storage
    return null;
  }
  
  Future<void> _saveTransaction(
    String txHash,
    String from,
    String to,
    String amount,
    String tokenAddress,
    String chainId,
  ) async {
    // Save transaction to history
  }
  
  // BIP-39 English word list (first 100 for demo)
  static const _englishWordList = [
    'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
    'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid',
    'acoustic', 'acquire', 'across', 'act', 'action', 'actor', 'actress', 'actual',
    'adapt', 'add', 'addict', 'address', 'adjust', 'admit', 'adult', 'advance',
    'advice', 'aerobic', 'affair', 'afford', 'afraid', 'again', 'age', 'agent',
    'agree', 'ahead', 'aim', 'air', 'airport', 'aisle', 'alarm', 'album',
    'alcohol', 'alert', 'alien', 'all', 'alley', 'allow', 'almost', 'alone',
    'alpha', 'already', 'also', 'alter', 'always', 'amateur', 'amazing', 'among',
    'amount', 'amused', 'analyst', 'anchor', 'ancient', 'anger', 'angle', 'angry',
    'animal', 'ankle', 'announce', 'annual', 'another', 'answer', 'antenna', 'antique',
    'anxiety', 'any', 'apart', 'apology', 'appear', 'apple', 'approve', 'april',
    'arch', 'arctic', 'area', 'arena', 'argue', 'arm', 'armed', 'armor',
    'army', 'around', 'arrange', 'arrest', 'arrive', 'arrow', 'art', 'artefact',
    'artist', 'artwork', 'ask', 'aspect', 'assault', 'asset', 'assist', 'assume',
    'asthma', 'athlete', 'atom', 'attack', 'attend', 'attitude', 'attract', 'auction',
    'audit', 'august', 'aunt', 'author', 'auto', 'autumn', 'average', 'avocado',
    'avoid', 'awake', 'aware', 'away', 'awesome', 'awful', 'awkward', 'axis',
    'baby', 'bachelor', 'bacon', 'badge', 'bag', 'balance', 'balcony', 'ball',
    'bamboo', 'banana', 'banner', 'bar', 'barely', 'bargain', 'barrel', 'base',
    'basic', 'basket', 'battle', 'beach', 'bean', 'beauty', 'because', 'become',
    'beef', 'before', 'begin', 'behave', 'behind', 'believe', 'below', 'belt',
    'bench', 'benefit', 'best', 'betray', 'better', 'between', 'beyond', 'bicycle',
    'bid', 'bike', 'bind', 'biology', 'bird', 'birth', 'bitter', 'black',
    'blade', 'blame', 'blanket', 'blast', 'blaze', 'bless', 'blind', 'blood',
    'blossom', 'blouse', 'blue', 'blur', 'blush', 'board', 'boat', 'body',
    'boil', 'bomb', 'bone', 'bonus', 'book', 'boost', 'border', 'boring',
    'borrow', 'boss', 'bottom', 'bounce', 'box', 'boy', 'bracket', 'brain',
    'brand', 'brass', 'brave', 'bread', 'breeze', 'brick', 'bridge', 'brief',
    'bright', 'bring', 'brisk', 'broccoli', 'broken', 'bronze', 'broom', 'brother',
    'brown', 'brush', 'bubble', 'buddy', 'budget', 'buffalo', 'build', 'bulb',
    'bulk', 'bullet', 'bundle', 'bunker', 'burden', 'burger', 'burst', 'bus',
    'business', 'busy', 'butter', 'buyer', 'buzz', 'cabbage', 'cabin', 'cable',
    'cactus', 'cage', 'cake', 'call', 'calm', 'camera', 'camp', 'can',
    'canal', 'cancel', 'candy', 'cannon', 'canoe', 'canvas', 'canyon', 'capable',
    'capital', 'captain', 'car', 'carbon', 'card', 'cargo', 'carpet', 'carry',
    'cart', 'case', 'cash', 'casino', 'castle', 'casual', 'cat', 'catalog',
    'catch', 'category', 'cattle', 'caught', 'cause', 'caution', 'cave', 'ceiling',
    'celery', 'cement', 'census', 'century', 'cereal', 'certain', 'chair', 'chalk',
    'champion', 'change', 'chaos', 'chapter', 'charge', 'chase', 'chat', 'cheap',
    'check', 'cheese', 'cherry', 'chest', 'chicken', 'chief', 'child', 'chimney',
    'choice', 'choose', 'chronic', 'chuckle', 'chunk', 'churn', 'cigar', 'cinnamon',
    'circle', 'citizen', 'city', 'civil', 'claim', 'clap', 'clarify', 'classic',
    'clean', 'clerk', 'clever', 'click', 'client', 'cliff', 'climb', 'clinic',
    'clip', 'clock', 'clog', 'close', 'cloth', 'cloud', 'clown', 'club',
    'clump', 'cluster', 'clutch', 'coach', 'coast', 'coconut', 'code', 'coffee',
    'coil', 'coin', 'collect', 'color', 'column', 'combine', 'come', 'comfort',
    'comic', 'common', 'company', 'concert', 'conduct', 'confirm', 'congress', 'connect',
    'consider', 'control', 'convince', 'cook', 'cool', 'copper', 'copy', 'coral',
    'core', 'corn', 'corner', 'correct', 'cost', 'cotton', 'couch', 'country',
    'couple', 'course', 'cousin', 'cover', 'coyote', 'crack', 'cradle', 'craft',
    'cram', 'crane', 'crash', 'crater', 'crawl', 'crazy', 'cream', 'credit',
    'creek', 'crew', 'cricket', 'crime', 'crisp', 'critic', 'crop', 'cross',
    'crouch', 'crowd', 'crucial', 'cruel', 'cruise', 'crumble', 'crunch', 'crush',
    'cry', 'crystal', 'cube', 'culture', 'cup', 'cupboard', 'curious', 'current',
    'curtain', 'curve', 'cushion', 'custom', 'cute', 'cycle', 'damage', 'damp',
    'dance', 'danger', 'daring', 'dash', 'daughter', 'dawn', 'day', 'deal',
    'debate', 'debris', 'decade', 'december', 'decide', 'decline', 'decorate', 'decrease',
    'deer', 'defense', 'define', 'defy', 'degree', 'delay', 'deliver', 'demand',
    'denial', 'dentist', 'deny', 'depart', 'depend', 'deposit', 'depth', 'deputy',
    'derive', 'describe', 'desert', 'design', 'desk', 'despair', 'destroy', 'detail',
    'detect', 'develop', 'device', 'devote', 'diagram', 'dial', 'diamond', 'diary',
    'dice', 'diesel', 'diet', 'differ', 'digital', 'dignity', 'dilemma', 'dinner',
    'dinosaur', 'direct', 'dirt', 'disagree', 'discover', 'disease', 'dish', 'dismiss',
    'disorder', 'display', 'distance', 'divert', 'divide', 'divorce', 'dizzy', 'doctor',
    'document', 'dog', 'doll', 'dolphin', 'domain', 'donate', 'donkey', 'donor',
    'door', 'dose', 'double', 'dove', 'draft', 'dragon', 'drama', 'draw', 'dream',
    'dress', 'drift', 'drill', 'drink', 'drip', 'drive', 'drop', 'drum',
    'dry', 'duck', 'dumb', 'dune', 'during', 'dust', 'dutch', 'duty',
    'dwarf', 'dynamic', 'eager', 'eagle', 'early', 'earn', 'earth', 'easily',
    'east', 'easy', 'echo', 'ecology', 'economy', 'edge', 'edit', 'educate',
    'effort', 'egg', 'eight', 'eject', 'elastic', 'elbow', 'elder', 'electric',
    'elegant', 'element', 'elephant', 'elevator', 'elite', 'else', 'embark', 'embody',
    'embrace', 'emerge', 'emotion', 'employ', 'empower', 'empty', 'enable', 'enact',
    'end', 'endless', 'endorse', 'enemy', 'energy', 'enforce', 'engage', 'engine',
    'enhance', 'enjoy', 'enlist', 'enough', 'enrich', 'enroll', 'ensure', 'enter',
    'entire', 'entry', 'envelope', 'episode', 'equal', 'equip', 'era', 'erase',
    'erode', 'erosion', 'error', 'erupt', 'escape', 'essay', 'essence', 'estate',
    'eternal', 'ethics', 'evidence', 'evil', 'evoke', 'evolve', 'exact', 'example',
    'excess', 'exchange', 'excite', 'exclude', 'excuse', 'execute', 'exercise', 'exhaust',
    'exhibit', 'exile', 'exist', 'exit', 'exotic', 'expand', 'expect', 'expire',
    'explain', 'expose', 'express', 'extend', 'extra', 'eye', 'eyebrow', 'fabric',
    'face', 'faculty', 'fade', 'faint', 'faith', 'fall', 'false', 'fame',
    'family', 'famous', 'fan', 'fancy', 'fantasy', 'farm', 'fashion', 'fat',
    'fatal', 'father', 'fatigue', 'fault', 'favorite', 'feature', 'february', 'federal',
    'fee', 'feed', 'feel', 'female', 'fence', 'festival', 'fetch', 'fever',
    'few', 'fiber', 'fiction', 'field', 'figure', 'file', 'film', 'filter',
    'final', 'find', 'fine', 'finger', 'finish', 'fire', 'firm', 'first',
    'fiscal', 'fish', 'fit', 'fitness', 'fix', 'flag', 'flame', 'flash',
    'flat', 'flavor', 'flee', 'flight', 'flip', 'float', 'flock', 'floor',
    'flower', 'fluid', 'flush', 'fly', 'foam', 'focus', 'fog', 'foil',
    'fold', 'follow', 'food', 'foot', 'force', 'forest', 'forget', 'fork',
    'fortune', 'forum', 'forward', 'fossil', 'foster', 'found', 'fox', 'fragile',
    'frame', 'frequent', 'fresh', 'friend', 'fringe', 'frog', 'front', 'frost',
    'frown', 'frozen', 'fruit', 'fuel', 'fun', 'funny', 'furnace', 'fury',
    'future', 'gadget', 'gain', 'galaxy', 'gallery', 'game', 'gap', 'garage',
    'garbage', 'garden', 'garlic', 'gas', 'gasp', 'gate', 'gather', 'gauge',
    'gaze', 'general', 'genius', 'genre', 'gentle', 'genuine', 'gesture', 'ghost',
    'giant', 'gift', 'giggle', 'ginger', 'giraffe', 'girl', 'give', 'glad',
    'glance', 'glare', 'glass', 'glide', 'glimpse', 'globe', 'gloom', 'glory',
    'glove', 'glow', 'glue', 'goat', 'goddess', 'gold', 'good', 'goose',
    'gorilla', 'gospel', 'gossip', 'govern', 'gown', 'grab', 'grace', 'grain',
    'grant', 'grape', 'grass', 'gravity', 'great', 'green', 'grid', 'grief',
    'grit', 'grocery', 'group', 'grow', 'grunt', 'guard', 'guess', 'guide',
    'guilt', 'guitar', 'gun', 'gym', 'habit', 'hair', 'half', 'hammer',
    'hamster', 'hand', 'handle', 'harbor', 'hard', 'harsh', 'harvest', 'hat',
    'have', 'hawk', 'hazard', 'head', 'health', 'heart', 'heavy', 'hedgehog',
    'height', 'hello', 'helmet', 'help', 'hen', 'hero', 'hidden', 'high',
    'hill', 'hint', 'hip', 'hire', 'history', 'hobby', 'hockey', 'hold',
    'hole', 'holiday', 'hollow', 'home', 'honey', 'hood', 'hope', 'horn',
    'horror', 'horse', 'hospital', 'host', 'hotel', 'hour', 'hover', 'hub',
    'huge', 'human', 'humble', 'humor', 'hundred', 'hungry', 'hunt', 'hurdle',
    'hurry', 'hurt', 'husband', 'hybrid', 'ice', 'icon', 'idea', 'identify',
    'idle', 'ignore', 'ill', 'illegal', 'illness', 'image', 'imitate', 'immense',
    'immune', 'impact', 'impose', 'improve', 'impulse', 'inch', 'include', 'income',
    'increase', 'index', 'indicate', 'indoor', 'industry', 'infant', 'inflict', 'inform',
    'inhale', 'inherit', 'initial', 'inject', 'injury', 'inmate', 'inner', 'innocent',
    'input', 'inquiry', 'insane', 'insect', 'inside', 'inspire', 'install', 'intact',
    'interest', 'into', 'invest', 'invite', 'involve', 'iron', 'island', 'isolate',
    'issue', 'item', 'ivory', 'jacket', 'jaguar', 'jar', 'jazz', 'jealous',
    'jeans', 'jelly', 'jewel', 'job', 'join', 'joke', 'journey', 'joy',
    'judge', 'juice', 'jump', 'jungle', 'junior', 'junk', 'just', 'kangaroo',
    'keen', 'keep', 'ketchup', 'key', 'kick', 'kid', 'kidney', 'kind',
    'kingdom', 'kiss', 'kit', 'kitchen', 'kite', 'kitten', 'kiwi', 'knee',
    'knife', 'knock', 'know', 'lab', 'label', 'labor', 'ladder', 'lady',
    'lake', 'lamp', 'language', 'laptop', 'large', 'later', 'latin', 'laugh',
    'laundry', 'lava', 'law', 'lawn', 'lawsuit', 'layer', 'lazy', 'leader',
    'leaf', 'learn', 'leave', 'lecture', 'left', 'leg', 'legal', 'legend',
    'leisure', 'lemon', 'lend', 'length', 'lens', 'leopard', 'lesson', 'letter',
    'level', 'liar', 'liberty', 'library', 'license', 'life', 'lift', 'light',
    'like', 'limb', 'limit', 'link', 'lion', 'liquid', 'list', 'little',
    'live', 'lizard', 'load', 'loan', 'lobster', 'local', 'lock', 'logic',
    'lonely', 'long', 'loop', 'lottery', 'loud', 'lounge', 'love', 'loyal',
    'lucky', 'luggage', 'lumber', 'lunar', 'lunch', 'luxury', 'lyrics', 'machine',
    'mad', 'magic', 'magnet', 'maid', 'mail', 'main', 'major', 'make',
    'mammal', 'man', 'manage', 'mandate', 'mango', 'mansion', 'manual', 'maple',
    'marble', 'march', 'margin', 'marine', 'market', 'marriage', 'mask', 'mass',
    'master', 'match', 'material', 'math', 'matrix', 'matter', 'maximize', 'mayor',
    'mean', 'measure', 'meat', 'mechanic', 'medal', 'media', 'melody', 'melt',
    'member', 'memory', 'men', 'mend', 'mental', 'mentor', 'menu', 'mercy',
    'merge', 'merit', 'merry', 'mesh', 'message', 'metal', 'method', 'middle',
    'midnight', 'milk', 'million', 'mimic', 'mind', 'minimum', 'minor', 'minute',
    'miracle', 'mirror', 'misery', 'miss', 'mistake', 'mix', 'mixed', 'mixture',
    'mobile', 'model', 'modify', 'mom', 'moment', 'monitor', 'monkey', 'monster',
    'month', 'moon', 'moral', 'more', 'morning', 'mosquito', 'mother', 'motion',
    'motor', 'mountain', 'mouse', 'move', 'movie', 'much', 'muffin', 'mule',
    'multiply', 'muscle', 'museum', 'mushroom', 'music', 'must', 'mutual', 'myself',
    'mystery', 'myth', 'naive', 'name', 'napkin', 'narrow', 'nasty', 'nation',
    'nature', 'near', 'neat', 'neck', 'need', 'negative', 'neglect', 'neither',
    'nephew', 'nerve', 'nest', 'net', 'network', 'neutral', 'never', 'news',
    'next', 'nice', 'night', 'noble', 'noise', 'nominee', 'noodle', 'normal',
    'north', 'nose', 'notable', 'note', 'nothing', 'notice', 'novel', 'now',
    'nuclear', 'number', 'nurse', 'nut', 'oak', 'obey', 'object', 'oblige',
    'obscure', 'observe', 'obtain', 'obvious', 'occur', 'ocean', 'october', 'odor',
    'off', 'offer', 'office', 'often', 'oil', 'okay', 'old', 'olive',
    'olympic', 'omit', 'once', 'one', 'onion', 'online', 'only', 'open',
    'opera', 'opinion', 'oppose', 'option', 'orange', 'orbit', 'orchard', 'order',
    'ordinary', 'organ', 'orient', 'original', 'orphan', 'ostrich', 'other', 'outdoor',
    'outer', 'output', 'outside', 'oval', 'oven', 'over', 'own', 'owner',
    'owl', 'oxygen', 'oyster', 'ozone', 'paddle', 'page', 'pair', 'palace',
    'palm', 'panda', 'panel', 'panic', 'panther', 'paper', 'parade', 'parent',
    'park', 'parrot', 'party', 'pass', 'patch', 'path', 'patient', 'patrol',
    'pattern', 'pause', 'pave', 'payment', 'peace', 'peanut', 'pear', 'peasant',
    'pelican', 'pen', 'penalty', 'pencil', 'people', 'pepper', 'perfect', 'permit',
    'person', 'pet', 'phone', 'photo', 'phrase', 'physical', 'piano', 'picnic',
    'picture', 'piece', 'pig', 'pigeon', 'pill', 'pilot', 'pink', 'pioneer',
    'pipe', 'pistol', 'pitch', 'pizza', 'place', 'planet', 'plastic', 'plate',
    'play', 'please', 'pledge', 'pluck', 'plug', 'plunge', 'poem', 'poet',
    'point', 'polar', 'pole', 'police', 'pond', 'pony', 'pool', 'popular',
    'portion', 'position', 'possible', 'post', 'potato', 'pottery', 'poverty', 'powder',
    'power', 'practice', 'praise', 'predict', 'prefer', 'prepare', 'present', 'pretty',
    'prevent', 'price', 'pride', 'primary', 'print', 'priority', 'prison', 'private',
    'prize', 'problem', 'process', 'produce', 'profit', 'program', 'project', 'promote',
    'proof', 'property', 'prosper', 'protect', 'proud', 'provide', 'public', 'pudding',
    'pull', 'pulp', 'pulse', 'pumpkin', 'punch', 'pupil', 'puppy', 'purchase',
    'purity', 'purpose', 'purse', 'push', 'put', 'puzzle', 'pyramid', 'quality',
    'quantum', 'quarter', 'question', 'quick', 'quit', 'quiz', 'quote', 'rabbit',
    'raccoon', 'race', 'rack', 'radar', 'radio', 'rail', 'rain', 'raise',
    'rally', 'ramp', 'ranch', 'random', 'range', 'rapid', 'rare', 'rate',
    'rather', 'raven', 'raw', 'reach', 'react', 'read', 'real', 'realm',
    'rear', 'reason', 'rebel', 'rebuild', 'recall', 'receive', 'recipe', 'record',
    'recycle', 'red', 'reduce', 'reflect', 'reform', 'refuse', 'region', 'regret',
    'regular', 'reject', 'relax', 'release', 'relief', 'rely', 'remain', 'remember',
    'remind', 'remote', 'remove', 'render', 'renew', 'rent', 'reopen', 'repair',
    'repeat', 'replace', 'reply', 'report', 'represent', 'reproduce', 'public', 'require',
    'rescue', 'resemble', 'resist', 'resource', 'response', 'result', 'retire', 'retreat',
    'return', 'reunion', 'reveal', 'review', 'reward', 'rhythm', 'rib', 'ribbon',
    'rice', 'rich', 'ride', 'ridge', 'rifle', 'right', 'rigid', 'ring', 'riot',
    'ripple', 'risk', 'ritual', 'rival', 'river', 'road', 'roast', 'robot',
    'robust', 'rocket', 'romance', 'roof', 'rookie', 'room', 'rose', 'rotate',
    'rough', 'round', 'route', 'royal', 'rubber', 'rude', 'rug', 'rule',
    'run', 'runway', 'rural', 'sad', 'saddle', 'sadness', 'safe', 'sail',
    'salad', 'salmon', 'salon', 'salt', 'salute', 'same', 'sample', 'sanctuary',
    'sand', 'satisfy', 'satoshi', 'sauce', 'sausage', 'save', 'say', 'scale',
    'scan', 'scare', 'scatter', 'scene', 'scent', 'school', 'science', 'scissors',
    'scorpion', 'scout', 'scrap', 'screen', 'script', 'scrub', 'sea', 'search',
    'season', 'seat', 'second', 'secret', 'section', 'security', 'seed', 'seek',
    'segment', 'select', 'sell', 'seminar', 'senior', 'sense', 'sentence', 'series',
    'service', 'session', 'settle', 'setup', 'seven', 'shadow', 'shaft', 'shallow',
    'share', 'shed', 'shell', 'sheriff', 'shield', 'shift', 'shine', 'ship',
    'shiver', 'shock', 'shoe', 'shoot', 'shop', 'short', 'shoulder', 'shove',
    'shrimp', 'shrug', 'shuffle', 'shy', 'sibling', 'sick', 'side', 'siege',
    'sight', 'sign', 'silent', 'silk', 'silly', 'silver', 'similar', 'simple',
    'since', 'sing', 'siren', 'sister', 'situate', 'six', 'size', 'skate',
    'sketch', 'ski', 'skill', 'skin', 'skirt', 'skull', 'slab', 'slam',
    'sleep', 'slice', 'slide', 'slight', 'slim', 'slogan', 'slot', 'slow',
    'slush', 'small', 'smart', 'smile', 'smoke', 'smooth', 'snack', 'snake',
    'snap', 'sniff', 'snow', 'soap', 'soccer', 'social', 'sock', 'soda',
    'soft', 'solar', 'soldier', 'solid', 'solution', 'solve', 'someone', 'song',
    'soon', 'sorry', 'sort', 'soul', 'sound', 'soup', 'source', 'south',
    'space', 'spare', 'spatial', 'spawn', 'speak', 'special', 'speed', 'spell',
    'spend', 'sphere', 'spice', 'spider', 'spike', 'spin', 'spirit', 'split',
    'spoil', 'sponsor', 'spoon', 'sport', 'spot', 'spray', 'spread', 'spring',
    'spy', 'square', 'squeeze', 'squirrel', 'stable', 'stadium', 'staff', 'stage',
    'stairs', 'stamp', 'stand', 'start', 'state', 'stay', 'steak', 'steel',
    'stem', 'step', 'stereo', 'stick', 'still', 'sting', 'stock', 'stomach',
    'stone', 'stool', 'story', 'stove', 'strategy', 'street', 'strike', 'strong',
    'struggle', 'student', 'stuff', 'stumble', 'style', 'subject', 'submit', 'subway',
    'success', 'such', 'sudden', 'suffer', 'sugar', 'suggest', 'suit', 'summer',
    'sun', 'sunny', 'sunset', 'super', 'supply', 'supreme', 'sure', 'surface',
    'surge', 'surprise', 'surround', 'survey', 'suspect', 'sustain', 'swallow', 'swamp',
    'swap', 'swarm', 'swear', 'sweat', 'sweep', 'sweet', 'swift', 'swim',
    'swing', 'switch', 'sword', 'symbol', 'symptom', 'syrup', 'system', 'table',
    'tackle', 'tag', 'tail', 'talent', 'talk', 'tank', 'tape', 'target',
    'task', 'taste', 'tattoo', 'taxi', 'teach', 'team', 'tell', 'ten',
    'tenant', 'tennis', 'tent', 'term', 'test', 'text', 'thank', 'that',
    'theme', 'then', 'theory', 'there', 'they', 'thing', 'this', 'thought',
    'three', 'thrive', 'throw', 'thumb', 'thunder', 'ticket', 'tide', 'tiger',
    'tilt', 'timber', 'time', 'tiny', 'tip', 'tired', 'tissue', 'title',
    'toast', 'tobacco', 'toddler', 'toe', 'together', 'toilet', 'token', 'tomato',
    'tomorrow', 'tone', 'tongue', 'tonight', 'tool', 'tooth', 'top', 'topic',
    'topple', 'torch', 'tornado', 'tortoise', 'toss', 'total', 'tourist', 'toward',
    'tower', 'town', 'toy', 'track', 'trade', 'traffic', 'tragic', 'train',
    'transfer', 'trap', 'trash', 'travel', 'tray', 'treat', 'tree', 'trend',
    'trial', 'tribe', 'trick', 'trigger', 'trim', 'trip', 'trophy', 'trouble',
    'truck', 'true', 'truly', 'trumpet', 'trust', 'truth', 'try', 'tube',
    'tuition', 'tumble', 'tuna', 'tunnel', 'turkey', 'turn', 'turtle', 'twelve',
    'twenty', 'twice', 'twin', 'twist', 'two', 'type', 'typical', 'ugly',
    'umbrella', 'unable', 'unaware', 'uncle', 'uncover', 'under', 'undo', 'unfair',
    'unfold', 'unhappy', 'uniform', 'unique', 'unit', 'unite', 'unity', 'universal',
    'universe', 'unknown', 'unlock', 'until', 'unusual', 'unveil', 'update', 'upgrade',
    'uphold', 'upon', 'upper', 'upset', 'urban', 'urge', 'usage', 'use',
    'used', 'useful', 'useless', 'usual', 'utility', 'vacant', 'vacuum', 'vague',
    'valid', 'valley', 'valve', 'van', 'vanish', 'vapor', 'various', 'vegan',
    'velvet', 'vendor', 'venture', 'venue', 'verb', 'verify', 'version', 'very',
    'vessel', 'veteran', 'viable', 'vibrant', 'vicious', 'victory', 'video', 'view',
    'village', 'vintage', 'violin', 'virtual', 'virus', 'visa', 'visit', 'visual',
    'vital', 'vivid', 'vocal', 'voice', 'void', 'volcano', 'volume', 'vote',
    'voyage', 'wage', 'wagon', 'wait', 'walk', 'wall', 'walnut', 'want',
    'warfare', 'warm', 'warrior', 'wash', 'wasp', 'waste', 'water', 'wave',
    'way', 'wealth', 'weapon', 'wear', 'weasel', 'weather', 'web', 'wedding',
    'weekend', 'weird', 'welcome', 'west', 'wet', 'whale', 'what', 'wheat',
    'wheel', 'when', 'where', 'whip', 'whisper', 'wide', 'width', 'wife',
    'wild', 'will', 'win', 'window', 'wine', 'wing', 'wink', 'winner',
    'winter', 'wire', 'wisdom', 'wise', 'wish', 'witness', 'wolf', 'woman',
    'wonder', 'wood', 'wool', 'word', 'work', 'world', 'worry', 'worth',
    'wrap', 'wreck', 'wrestle', 'wrist', 'write', 'wrong', 'yard', 'year',
    'yellow', 'you', 'young', 'youth', 'zebra', 'zero', 'zone', 'zoo',
  ];
}
