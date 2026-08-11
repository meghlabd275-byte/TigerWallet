// DAO Service - Flutter Mobile
// Governance and DAO functionality

import 'dart:convert';
import 'package:http/http.dart' as http;

class DAOService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  DAOService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get DAOs
  Future<List<DAO>> getDAOs() async {
    final response = await http.get(
      Uri.parse('$API_BASE/dao/list'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((d) => DAO.fromJson(d)).toList();
    }
    return [];
  }
  
  // Get DAO proposals
  Future<List<Proposal>> getProposals(String daoId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/dao/$daoId/proposals'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => Proposal.fromJson(p)).toList();
    }
    return [];
  }
  
  // Create proposal
  Future<Proposal> createProposal(String daoId, String title, String description, String type) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dao/$daoId/proposals'),
      headers: _headers,
      body: json.encode({
        'title': title,
        'description': description,
        'type': type,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return Proposal.fromJson(data['data']);
    }
    throw Exception('Failed to create proposal');
  }
  
  // Vote
  Future<bool> vote(String proposalId, String vote, double weight) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dao/proposals/$proposalId/vote'),
      headers: _headers,
      body: json.encode({'vote': vote, 'weight': weight}),
    );
    
    return response.statusCode == 200;
  }
  
  // Execute proposal
  Future<bool> executeProposal(String proposalId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dao/proposals/$proposalId/execute'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Delegate voting power
  Future<bool> delegate(String daoId, String delegatee) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dao/$daoId/delegate'),
      headers: _headers,
      body: json.encode({'delegatee': delegatee}),
    );
    
    return response.statusCode == 200;
  }
}

class DAO {
  final String id;
  final String name;
  final String description;
  final String token;
  final String treasuryAddress;
  final int memberCount;
  final double treasuryValue;
  
  DAO({
    required this.id,
    required this.name,
    required this.description,
    required this.token,
    required this.treasuryAddress,
    required this.memberCount,
    required this.treasuryValue,
  });
  
  factory DAO.fromJson(Map<String, dynamic> json) {
    return DAO(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      description: json['description'] ?? '',
      token: json['token'] ?? '',
      treasuryAddress: json['treasuryAddress'] ?? '',
      memberCount: json['memberCount'] ?? 0,
      treasuryValue: (json['treasuryValue'] ?? 0).toDouble(),
    );
  }
}

class Proposal {
  final String id;
  final String daoId;
  final String title;
  final String description;
  final String type;
  final String status;
  final double forVotes;
  final double againstVotes;
  final int quorum;
  final DateTime createdAt;
  final DateTime? executedAt;
  
  Proposal({
    required this.id,
    required this.daoId,
    required this.title,
    required this.description,
    required this.type,
    required this.status,
    required this.forVotes,
    required this.againstVotes,
    required this.quorum,
    required this.createdAt,
    this.executedAt,
  });
  
  factory Proposal.fromJson(Map<String, dynamic> json) {
    return Proposal(
      id: json['id'] ?? '',
      daoId: json['daoId'] ?? '',
      title: json['title'] ?? '',
      description: json['description'] ?? '',
      type: json['type'] ?? '',
      status: json['status'] ?? 'PENDING',
      forVotes: (json['forVotes'] ?? 0).toDouble(),
      againstVotes: (json['againstVotes'] ?? 0).toDouble(),
      quorum: json['quorum'] ?? 0,
      createdAt: DateTime.parse(json['createdAt']),
      executedAt: json['executedAt'] != null ? DateTime.parse(json['executedAt']) : null,
    );
  }
}
