//
//  DomainModels.swift
//  TigerAdmin
//
//  Generic Codable wrapper for the 12 admin domain endpoints
//  (futures, options, copy-trading, convert, onramp, offramp, p2p-clients,
//  p2p-merchants, partners, rewards, marketing, roles, permissions).
//

import Foundation

// Loose wrapper: fields vary per domain, so all fields except id are optional.
// Decodes defensively (any numeric id is coerced to String) so a missing or
// differently-typed field never breaks the whole list.
struct DomainRecord: Codable, Identifiable {
    let id: String
    var name: String?
    var symbol: String?
    var status: String?
    var email: String?
    var type: String?
    var description: String?
    var resource: String?
    var action: String?
    var permissions: [String]?
    var trader: String?
    var followers: Int?
    var leverage: String?
    var margin: String?
    var strike: String?
    var expiry: String?
    var fromAsset: String?
    var toAsset: String?
    var amount: String?
    var rate: String?
    var user: String?
    var provider: String?
    var verified: Bool?
    var campaign: String?

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: AnyKey.self)
        func str(_ k: String) -> String? {
            if let v = try? c.decode(String.self, forKey: AnyKey(stringValue: k)) { return v }
            if let v = try? c.decode(Int.self, forKey: AnyKey(stringValue: k)) { return String(v) }
            if let v = try? c.decode(Double.self, forKey: AnyKey(stringValue: k)) { return String(v) }
            return nil
        }
        func bool(_ k: String) -> Bool? { try? c.decode(Bool.self, forKey: AnyKey(stringValue: k)) }
        func ints(_ k: String) -> [String]? { try? c.decode([String].self, forKey: AnyKey(stringValue: k)) }
        func intv(_ k: String) -> Int? { try? c.decode(Int.self, forKey: AnyKey(stringValue: k)) }
        id = str("id") ?? UUID().uuidString
        name = str("name"); symbol = str("symbol"); status = str("status")
        email = str("email"); type = str("type"); description = str("description")
        resource = str("resource"); action = str("action")
        permissions = ints("permissions")
        trader = str("trader"); followers = intv("followers")
        leverage = str("leverage"); margin = str("margin")
        strike = str("strike"); expiry = str("expiry")
        fromAsset = str("from_asset"); toAsset = str("to_asset")
        amount = str("amount"); rate = str("rate")
        user = str("user"); provider = str("provider")
        verified = bool("verified"); campaign = str("campaign")
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: AnyKey.self)
        try c.encode(id, forKey: AnyKey(stringValue: "id"))
        try c.encodeIfPresent(name, forKey: AnyKey(stringValue: "name"))
        try c.encodeIfPresent(status, forKey: AnyKey(stringValue: "status"))
    }
}

// Coding key that accepts arbitrary string keys (used by DomainRecord).
struct AnyKey: CodingKey {
    var stringValue: String
    init(stringValue: String) { self.stringValue = stringValue }
    init?(stringValue: String) { self.stringValue = stringValue }
    var intValue: Int? { nil }
    init?(intValue: Int) { nil }
}

// MARK: - New admin domains (bots, bots-clients, project-teams, liquidity-sources)
// Loose Codable structs mirroring the admin/go (port 9093) payloads. Fields are
// optional and decoded defensively so a partial/missing field never breaks a list.

struct BotDomainRecord: Codable, Identifiable {
    let id: String
    var name: String?
    var botType: String?
    var leverage: Int?
    var status: String?
    var createdAt: String?
    var updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id, name, status
        case botType = "bot_type"
        case leverage
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        func str(_ k: KeyedDecodingContainer<CodingKeys>.Key) -> String? {
            if let v = try? c.decode(String.self, forKey: k) { return v }
            if let v = try? c.decode(Int.self, forKey: k) { return String(v) }
            return nil
        }
        id = str(.id) ?? UUID().uuidString
        name = str(.name); botType = str(.botType); status = str(.status)
        leverage = try? c.decode(Int.self, forKey: .leverage)
        createdAt = str(.createdAt); updatedAt = str(.updatedAt)
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(id, forKey: .id)
        try c.encodeIfPresent(name, forKey: .name)
        try c.encodeIfPresent(botType, forKey: .botType)
        try c.encodeIfPresent(leverage, forKey: .leverage)
        try c.encodeIfPresent(status, forKey: .status)
    }
}

struct BotDomainRequest: Codable {
    var name: String
    var botType: String?
    var leverage: Int?
    enum CodingKeys: String, CodingKey { case name; case botType = "bot_type"; case leverage }
}

struct BotTierRecord: Codable, Identifiable {
    let id: String
    var name: String?
    var minVolume: String?
    var maxVolume: String?
    var feeRate: String?
    enum CodingKeys: String, CodingKey {
        case id, name
        case minVolume = "min_volume"; case maxVolume = "max_volume"; case feeRate = "fee_rate"
    }
}

struct BotTierRequest: Codable {
    var name: String?
    var minVolume: String?
    var maxVolume: String?
    var feeRate: String?
    enum CodingKeys: String, CodingKey {
        case name; case minVolume = "min_volume"; case maxVolume = "max_volume"; case feeRate = "fee_rate"
    }
}

struct BotsClientRecord: Codable, Identifiable {
    let id: String
    var name: String?
    var clientId: String?
    var apiKey: String?
    var status: String?
    var createdAt: String?
    enum CodingKeys: String, CodingKey {
        case id, name, status
        case clientId = "client_id"; case apiKey = "api_key"; case createdAt = "created_at"
    }
}

struct BotsClientRequest: Codable {
    var name: String
    var clientId: String?
    var apiKey: String?
    enum CodingKeys: String, CodingKey { case name; case clientId = "client_id"; case apiKey = "api_key" }
}

struct ProjectTeamRecord: Codable, Identifiable {
    let id: String
    var name: String?
    var description: String?
    var status: String?
    var createdAt: String?
    enum CodingKeys: String, CodingKey { case id, name, description, status; case createdAt = "created_at" }
}

struct ProjectTeamRequest: Codable {
    var name: String
    var description: String?
}

struct ProjectTeamMemberRecord: Codable, Identifiable {
    let id: String
    var teamId: String?
    var userId: String?
    var role: String?
    var status: String?
    enum CodingKeys: String, CodingKey {
        case id, role, status
        case teamId = "team_id"; case userId = "user_id"
    }
}

struct AddProjectTeamMemberRequest: Codable {
    var userId: String
    var role: String?
    enum CodingKeys: String, CodingKey { case userId = "user_id"; case role }
}

struct LiquiditySourceRecord: Codable, Identifiable {
    let id: String
    var name: String?
    var provider: String?
    var priority: Int?
    var status: String?
    var healthy: Bool?
    var lastCheck: String?
    enum CodingKeys: String, CodingKey {
        case id, name, provider, priority, status
        case healthy; case lastCheck = "last_check"
    }
}

struct LiquiditySourceRequest: Codable {
    var name: String
    var provider: String?
    var priority: Int?
}

struct SetLiquiditySourcePriorityRequest: Codable {
    let priority: Int
}

struct DomainStats: Codable {
    var total: Int?
    var active: Int?
    var inactive: Int?
}
