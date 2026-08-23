//
//  APIClient.swift
//  TigerWallet
//
//  Production-Ready API Client with Request/Response Handling
//

import Foundation

// MARK: - API Configuration

struct APIConfig {
    static let baseURL = "http://localhost:8443"
    // SEPARATION: UserWallet talks ONLY to its own backend (go/wallet_api :8443).
    // Multi-sig and all signing/approval are delegated server-side; the client
    // NEVER calls the MasterWallet backend (:8450) directly.
    static let timeout: TimeInterval = 30
    static let maxRetries = 3
}

// MARK: - HTTP Methods

enum HTTPMethod: String {
    case GET
    case POST
    case PUT
    case PATCH
    case DELETE
}

// MARK: - API Error

enum APIError: Error {
    case invalidURL
    case networkError(Error)
    case invalidResponse
    case httpError(statusCode: Int, message: String?)
    case decodingError(Error)
    case unauthorized
    case forbidden
    case notFound
    case serverError
    case rateLimited
    case unknown
}

// MARK: - API Client

class APIClient {
    static let shared = APIClient()
    
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    
    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = APIConfig.timeout
        config.timeoutIntervalForResource = APIConfig.timeout * 2
        config.waitsForConnectivity = true
        
        self.session = URLSession(configuration: config)
        
        self.decoder = JSONDecoder()
        self.decoder.dateDecodingStrategy = .iso8601
        self.decoder.keyDecodingStrategy = .convertFromSnakeCase
        
        self.encoder = JSONEncoder()
        self.encoder.dateEncodingStrategy = .iso8601
        self.encoder.keyEncodingStrategy = .convertToSnakeCase
    }
    
    // MARK: - GET Request
    
    func get<T: Decodable>(
        endpoint: String,
        parameters: [String: String]? = nil,
        authenticated: Bool = false
    ) async throws -> T {
        let request = try buildRequest(
            endpoint: endpoint,
            method: .GET,
            parameters: parameters,
            authenticated: authenticated
        )
        
        return try await performRequest(request)
    }
    
    // MARK: - POST Request
    
    func post<T: Decodable, B: Encodable>(
        endpoint: String,
        body: B,
        authenticated: Bool = false
    ) async throws -> T {
        let request = try buildRequest(
            endpoint: endpoint,
            method: .POST,
            body: body,
            authenticated: authenticated
        )
        
        return try await performRequest(request)
    }
    
    // MARK: - PUT Request
    
    func put<T: Decodable, B: Encodable>(
        endpoint: String,
        body: B,
        authenticated: Bool = false
    ) async throws -> T {
        let request = try buildRequest(
            endpoint: endpoint,
            method: .PUT,
            body: body,
            authenticated: authenticated
        )
        
        return try await performRequest(request)
    }
    
    // MARK: - PATCH Request
    
    func patch<T: Decodable, B: Encodable>(
        endpoint: String,
        body: B,
        authenticated: Bool = false
    ) async throws -> T {
        let request = try buildRequest(
            endpoint: endpoint,
            method: .PATCH,
            body: body,
            authenticated: authenticated
        )
        
        return try await performRequest(request)
    }
    
    // MARK: - DELETE Request
    
    func delete<T: Decodable>(
        endpoint: String,
        authenticated: Bool = false
    ) async throws -> T {
        let request = try buildRequest(
            endpoint: endpoint,
            method: .DELETE,
            authenticated: authenticated
        )
        
        return try await performRequest(request)
    }
    
    // MARK: - Build Request
    
    private func buildRequest<B: Encodable>(
        endpoint: String,
        method: HTTPMethod,
        parameters: [String: String]? = nil,
        body: B? = nil,
        authenticated: Bool = false
    ) throws -> URLRequest {
        var urlString = APIConfig.baseURL + endpoint
        
        // Add query parameters for GET requests
        if method == .GET, let parameters = parameters {
            let queryItems = parameters.map { URLQueryItem(name: $0.key, value: $0.value) }
            var components = URLComponents(string: urlString)
            components?.queryItems = queryItems
            urlString = components?.string ?? urlString
        }
        
        guard let url = URL(string: urlString) else {
            throw APIError.invalidURL
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("TigerWallet-iOS/1.0", forHTTPHeaderField: "User-Agent")
        
        // Add body for non-GET requests
        if let body = body {
            request.httpBody = try encoder.encode(body)
        }
        
        // Add authentication
        if authenticated {
            guard let token = AuthManager.shared.sessionToken else {
                throw APIError.unauthorized
            }
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        
        return request
    }
    
    // MARK: - Perform Request
    
    private func performRequest<T: Decodable>(_ request: URLRequest) async throws -> T {
        var lastError: Error?
        
        for attempt in 0..<APIConfig.maxRetries {
            do {
                let (data, response) = try await session.data(for: request)
                
                guard let httpResponse = response as? HTTPURLResponse else {
                    throw APIError.invalidResponse
                }
                
                // Handle HTTP status codes
                switch httpResponse.statusCode {
                case 200...299:
                    // Success - decode response
                    do {
                        return try decoder.decode(T.self, from: data)
                    } catch {
                        throw APIError.decodingError(error)
                    }
                    
                case 401:
                    // Unauthorized - try to refresh token
                    if attempt == 0 {
                        try await AuthManager.shared.refreshSession()
                        // Retry with new token
                        continue
                    }
                    throw APIError.unauthorized
                    
                case 403:
                    throw APIError.forbidden
                    
                case 404:
                    throw APIError.notFound
                    
                case 429:
                    throw APIError.rateLimited
                    
                case 500...599:
                    throw APIError.serverError
                    
                default:
                    // Try to decode error message
                    if let errorResponse = try? decoder.decode(ErrorResponse.self, from: data) {
                        throw APIError.httpError(statusCode: httpResponse.statusCode, message: errorResponse.message)
                    }
                    throw APIError.httpError(statusCode: httpResponse.statusCode, message: nil)
                }
                
            } catch let error as APIError {
                // Re-throw API errors
                throw error
            } catch {
                lastError = error
                
                // Exponential backoff for retries
                if attempt < APIConfig.maxRetries - 1 {
                    let delay = pow(2.0, Double(attempt))
                    try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
                }
            }
        }
        
        throw APIError.networkError(lastError ?? APIError.unknown)
    }
}

// MARK: - Error Response

struct ErrorResponse: Decodable {
    let code: Int?
    let message: String?
    let details: String?
}

// MARK: - Generic API Response

struct APIResponse<T: Decodable>: Decodable {
    let success: Bool
    let data: T?
    let error: ErrorResponse?
}
