//
//  NFTService.swift
//  TigerWallet
//
//  Complete NFT Service for iOS
//

import Foundation

class NFTService {
    static let shared = NFTService()
    private let baseURL = "https://api.tigerwallet.com/v1/nft"
    
    private init() {}
    
    // MARK: - Fetch NFTs
    
    func fetchNFTs(address: String, chain: String, completion: @escaping (Result<[NFT], Error>) -> Void) {
        let url = URL(string: "\(baseURL)/\(chain)/\(address)")!
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "NFTService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let nfts = try JSONDecoder().decode([NFT].self, from: data)
                completion(.success(nfts))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    // MARK: - Fetch Collections
    
    func fetchCollections(chain: String, completion: @escaping (Result<[NFTCollection], Error>) -> Void) {
        let url = URL(string: "\(baseURL)/collections/\(chain)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "NFTService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let collections = try JSONDecoder().decode([NFTCollection].self, from: data)
                completion(.success(collections))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    // MARK: - Transfer NFT
    
    func transferNFT(to: String, tokenId: String, contractAddress: String, chain: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/transfer")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "to": to,
            "tokenId": tokenId,
            "contractAddress": contractAddress,
            "chain": chain
        ]
        
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = json["txHash"] as? String else {
                completion(.failure(NSError(domain: "NFTService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Transfer failed"])))
                return
            }
            
            completion(.success(txHash))
        }.resume()
    }
    
    // MARK: - Fetch Listings
    
    func fetchListings(contractAddress: String, chain: String, completion: @escaping (Result<[NFTListing], Error>) -> Void) {
        let url = URL(string: "\(baseURL)/listings/\(chain)/\(contractAddress)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "NFTService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let listings = try JSONDecoder().decode([NFTListing].self, from: data)
                completion(.success(listings))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    // MARK: - Buy NFT
    
    func buyNFT(listingId: String, address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/buy")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["listingId": listingId, "address": address]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = json["txHash"] as? String else {
                completion(.failure(NSError(domain: "NFTService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Purchase failed"])))
                return
            }
            
            completion(.success(txHash))
        }.resume()
    }
    
    // MARK: - List NFT for Sale
    
    func listNFT(tokenId: String, contractAddress: String, price: String, chain: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/list")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "tokenId": tokenId,
            "contractAddress": contractAddress,
            "price": price,
            "chain": chain
        ]
        
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let listingId = json["listingId"] as? String else {
                completion(.failure(NSError(domain: "NFTService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Listing failed"])))
                return
            }
            
            completion(.success(listingId))
        }.resume()
    }
}

// MARK: - Models

struct NFT: Codable {
    let id: String
    let tokenId: String
    let contractAddress: String
    let name: String
    let description: String?
    let imageURL: String?
    let animationURL: String?
    let owner: String
    let chain: String
    
    enum CodingKeys: String, CodingKey {
        case id, tokenId, contractAddress, name, description, imageURL = "imageUrl", animationURL = "animationUrl", owner, chain
    }
}

struct NFTCollection: Codable {
    let address: String
    let name: String
    let description: String?
    let imageURL: String?
    let floorPrice: Double
    let totalSupply: Int
    
    enum CodingKeys: String, CodingKey {
        case address, name, description, imageURL = "imageUrl", floorPrice = "floorPrice", totalSupply = "totalSupply"
    }
}

struct NFTListing: Codable {
    let id: String
    let tokenId: String
    let contractAddress: String
    let seller: String
    let price: String
    let chain: String
}
