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
