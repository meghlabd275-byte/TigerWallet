//
//  NFTMarketplaceService.swift
//  TigerWallet
//
//  Production-ready NFT marketplace functionality
//  Buy, sell, create listings, and trade NFTs
//

import Foundation

// MARK: - NFT Models

struct NFTCollection: Codable, Identifiable {
    let id: String
    let address: String
    let name: String
    let symbol: String
    let description: String?
    let imageUrl: String?
    let bannerUrl: String?
    let floorPrice: Double
    let totalSupply: Int
    let owners: Int
    let volume24h: Double
    let chain: String
}

struct NFTListing: Codable, Identifiable {
    let id: String
    let tokenId: String
    let collectionAddress: String
    let seller: String
    let price: Double
    let priceToken: String
    let expirationTime: Int
    let chain: String
    let imageUrl: String?
    let name: String?
}

struct NFTItem: Codable, Identifiable {
    let id: String
    let tokenId: String
    let collectionAddress: String
    let name: String
    let description: String?
    let imageUrl: String?
    let animationUrl: String?
    let attributes: [NFTAttribute]?
    let owner: String?
    let price: Double?
    let chain: String
}

struct NFTAttribute: Codable {
    let traitType: String
    let value: String
}

struct NFTTrade: Codable, Identifiable {
    let id: String
    let tokenId: String
    let collectionAddress: String
    let buyer: String
    let seller: String
    let price: Double
    let timestamp: Int
    let txHash: String
    let chain: String
}

// MARK: - Marketplace Service

class NFTMarketplaceService {
    
    static let shared = NFTMarketplaceService()
    
    private let session = URLSession.shared
    
    // MARK: - Collection Methods
    
    /// Get collection by address
    func getCollection(address: String, chain: String = "ethereum") async throws -> NFTCollection? {
        let url = URL(string: "http://localhost:8443/api/v1/nft/collections/\(address)?chain=\(chain)")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return nil
        }
        
        let collection = try JSONDecoder().decode(NFTCollection.self, from: data)
        return collection
    }
    
    /// Search collections
    func searchCollections(query: String, chain: String = "ethereum", limit: Int = 20) async throws -> [NFTCollection] {
        var components = URLComponents(string: "http://localhost:8443/api/v1/nft/collections/search")!
        components.queryItems = [
            URLQueryItem(name: "q", value: query),
            URLQueryItem(name: "chain", value: chain),
            URLQueryItem(name: "limit", value: String(limit))
        ]
        
        let (data, response) = try await session.data(from: components.url!)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }
        
        let collections = try JSONDecoder().decode([NFTCollection].self, from: data)
        return collections
    }
    
    // MARK: - NFT Methods
    
    /// Get NFTs for a collection
    func getNFTs(collectionAddress: String, chain: String = "ethereum", limit: Int = 50, offset: Int = 0) async throws -> [NFTItem] {
        var components = URLComponents(string: "http://localhost:8443/api/v1/nft/assets")!
        components.queryItems = [
            URLQueryItem(name: "collection", value: collectionAddress),
            URLQueryItem(name: "chain", value: chain),
            URLQueryItem(name: "limit", value: String(limit)),
            URLQueryItem(name: "offset", value: String(offset))
        ]
        
        let (data, response) = try await session.data(from: components.url!)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }
        
        let nfts = try JSONDecoder().decode([NFTItem].self, from: data)
        return nfts
    }
    
    /// Get NFTs for a wallet
    func getUserNFTs(walletAddress: String, chain: String = "ethereum") async throws -> [NFTItem] {
        let url = URL(string: "http://localhost:8443/api/v1/nft/owners/\(walletAddress)?chain=\(chain)")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }
        
        let nfts = try JSONDecoder().decode([NFTItem].self, from: data)
        return nfts
    }
    
    // MARK: - Listing Methods
    
    /// Get listings for a collection
    func getListings(collectionAddress: String, chain: String = "ethereum", limit: Int = 50) async throws -> [NFTListing] {
        var components = URLComponents(string: "http://localhost:8443/api/v1/nft/listings")!
        components.queryItems = [
            URLQueryItem(name: "collection", value: collectionAddress),
            URLQueryItem(name: "chain", value: chain),
            URLQueryItem(name: "limit", value: String(limit))
        ]
        
        let (data, response) = try await session.data(from: components.url!)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }
        
        let listings = try JSONDecoder().decode([NFTListing].self, from: data)
        return listings
    }
    
    /// Get user's listings
    func getUserListings(walletAddress: String, chain: String = "ethereum") async throws -> [NFTListing] {
        let url = URL(string: "http://localhost:8443/api/v1/nft/listings/\(walletAddress)?chain=\(chain)")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }
        
        let listings = try JSONDecoder().decode([NFTListing].self, from: data)
        return listings
    }
    
    // MARK: - Trading Methods
    
    /// Create a listing (sell NFT)
    func createListing(
        walletAddress: String,
        collectionAddress: String,
        tokenId: String,
        price: Double,
        priceToken: String = "ETH",
        chain: String = "ethereum"
    ) async throws -> String {
        let url = URL(string: "http://localhost:8443/api/v1/nft/listings")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "collection_address": collectionAddress,
            "token_id": tokenId,
            "price": price,
            "price_token": priceToken,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw NFTError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(ListingResult.self, from: data)
        return result.listingId
    }
    
    /// Cancel a listing
    func cancelListing(walletAddress: String, listingId: String, chain: String = "ethereum") async throws -> Bool {
        let url = URL(string: "http://localhost:8443/api/v1/nft/listings/\(listingId)")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (_, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw NFTError.transactionFailed
        }
        
        return true
    }
    
    /// Buy NFT (fulfill listing)
    func buyNFT(buyerAddress: String, listingId: String, chain: String = "ethereum") async throws -> String {
        let url = URL(string: "http://localhost:8443/api/v1/nft/purchase")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "buyer_address": buyerAddress,
            "listing_id": listingId,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw NFTError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    // MARK: - Offer Methods
    
    /// Make an offer
    func makeOffer(
        makerAddress: String,
        collectionAddress: String,
        tokenId: String,
        price: Double,
        priceToken: String = "ETH",
        expirationTime: Int,
        chain: String = "ethereum"
    ) async throws -> String {
        let url = URL(string: "http://localhost:8443/api/v1/nft/offers")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "maker_address": makerAddress,
            "collection_address": collectionAddress,
            "token_id": tokenId,
            "price": price,
            "price_token": priceToken,
            "expiration_time": expirationTime,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw NFTError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(OfferResult.self, from: data)
        return result.offerId
    }
    
    /// Accept an offer
    func acceptOffer(sellerAddress: String, offerId: String, chain: String = "ethereum") async throws -> String {
        let url = URL(string: "http://localhost:8443/api/v1/nft/offers/\(offerId)/accept")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "seller_address": sellerAddress,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw NFTError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    // MARK: - Trade History
    
    /// Get trade history for a collection
    func getCollectionTrades(collectionAddress: String, chain: String = "ethereum", limit: Int = 50) async throws -> [NFTTrade] {
        var components = URLComponents(string: "http://localhost:8443/api/v1/nft/trades")!
        components.queryItems = [
            URLQueryItem(name: "collection", value: collectionAddress),
            URLQueryItem(name: "chain", value: chain),
            URLQueryItem(name: "limit", value: String(limit))
        ]
        
        let (data, response) = try await session.data(from: components.url!)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }
        
        let trades = try JSONDecoder().decode([NFTTrade].self, from: data)
        return trades
    }
}

// MARK: - Error Types

enum NFTError: Error {
    case networkError
    case transactionFailed
    case invalidParameters
    case insufficientBalance
}

// MARK: - Result Types

struct ListingResult: Codable {
    let listingId: String
}

struct OfferResult: Codable {
    let offerId: String
}

struct TransactionResult: Codable {
    let txHash: String
}

// MARK: - Marketplace Enum

enum NFTMarketplace: String, CaseIterable {
    case opensea = "OpenSea"
    case looksRare = "LooksRare"
    case x2y2 = "X2Y2"
    case blur = "Blur"
    
    var displayName: String {
        return self.rawValue
    }
}
