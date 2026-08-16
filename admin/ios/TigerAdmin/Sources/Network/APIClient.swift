/**
 * TigerWallet Admin - API Client
 */

import Foundation
import Alamofire

class APIClient {
    static let shared = APIClient()
    
    private let baseURL = "http://localhost:9093/api/v1"
    private let session: Session
    
    private init() {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 30
        configuration.timeoutIntervalForResource = 60
        
        session = Session(configuration: configuration)
    }
    
    private var headers: HTTPHeaders {
        var headers: HTTPHeaders = [
            "Content-Type": "application/json",
            "Accept": "application/json"
        ]
        
        if let token = AuthService.shared.token {
            headers["Authorization"] = "Bearer \(token)"
        }
        
        return headers
    }
    
    // MARK: - Auth
    
    func login(email: String, password: String, completion: @escaping (Result<LoginResponse, Error>) -> Void) {
        let parameters: [String: Any] = [
            "email": email,
            "password": password
        ]
        
        session.request("\(baseURL)/auth/login", method: .post, parameters: parameters, encoding: JSONEncoding.default, headers: headers)
            .validate()
            .responseDecodable(of: LoginResponse.self) { response in
                switch response.result {
                case .success(let data):
                    completion(.success(data))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    // MARK: - Users
    
    func getUsers(page: Int = 1, limit: Int = 20, completion: @escaping (Result<[User], Error>) -> Void) {
        let parameters: [String: Any] = ["page": page, "limit": limit]
        
        session.request("\(baseURL)/users", method: .get, parameters: parameters, headers: headers)
            .validate()
            .responseDecodable(of: UsersResponse.self) { response in
                switch response.result {
                case .success(let data):
                    completion(.success(data.users))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    func getUser(id: String, completion: @escaping (Result<User, Error>) -> Void) {
        session.request("\(baseURL)/users/\(id)", method: .get, headers: headers)
            .validate()
            .responseDecodable(of: User.self) { response in
                switch response.result {
                case .success(let user):
                    completion(.success(user))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    func updateUserStatus(id: String, status: User.UserStatus, completion: @escaping (Result<User, Error>) -> Void) {
        let parameters: [String: Any] = ["status": status.rawValue]
        
        session.request("\(baseURL)/users/\(id)/status", method: .put, parameters: parameters, encoding: JSONEncoding.default, headers: headers)
            .validate()
            .responseDecodable(of: User.self) { response in
                switch response.result {
                case .success(let user):
                    completion(.success(user))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    // MARK: - KYC
    
    func getKYCRequests(status: String? = nil, page: Int = 1, completion: @escaping (Result<[KycRequest], Error>) -> Void) {
        var parameters: [String: Any] = ["page": page]
        if let status = status {
            parameters["status"] = status
        }
        
        session.request("\(baseURL)/kyc", method: .get, parameters: parameters, headers: headers)
            .validate()
            .responseDecodable(of: KYCResponse.self) { response in
                switch response.result {
                case .success(let data):
                    completion(.success(data.requests))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    func reviewKYC(id: String, approved: Bool, completion: @escaping (Result<KycRequest, Error>) -> Void) {
        let parameters: [String: Any] = ["approved": approved]
        
        session.request("\(baseURL)/kyc/\(id)/review", method: .post, parameters: parameters, encoding: JSONEncoding.default, headers: headers)
            .validate()
            .responseDecodable(of: KycRequest.self) { response in
                switch response.result {
                case .success(let request):
                    completion(.success(request))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    // MARK: - Transactions
    
    func getTransactions(page: Int = 1, limit: Int = 20, completion: @escaping (Result<[Transaction], Error>) -> Void) {
        let parameters: [String: Any] = ["page": page, "limit": limit]
        
        session.request("\(baseURL)/transactions", method: .get, parameters: parameters, headers: headers)
            .validate()
            .responseDecodable(of: TransactionsResponse.self) { response in
                switch response.result {
                case .success(let data):
                    completion(.success(data.transactions))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    func flagTransaction(id: String, reason: String, completion: @escaping (Result<Transaction, Error>) -> Void) {
        let parameters: [String: Any] = ["reason": reason]
        
        session.request("\(baseURL)/transactions/\(id)/flag", method: .post, parameters: parameters, encoding: JSONEncoding.default, headers: headers)
            .validate()
            .responseDecodable(of: Transaction.self) { response in
                switch response.result {
                case .success(let tx):
                    completion(.success(tx))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
    
    // MARK: - Tokens
    
    func getTokens(completion: @escaping (Result<[Token], Error>) -> Void) {
        session.request("\(baseURL)/tokens", method: .get, headers: headers)
            .validate()
            .responseDecodable(of: TokensResponse.self) { response in
                switch response.result {
                case .success(let data):
                    completion(.success(data.tokens))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
    }
}

// MARK: - Response Models

struct LoginResponse: Codable {
    let token: String
    let refreshToken: String
    let user: User
}

struct UsersResponse: Codable {
    let users: [User]
    let total: Int
    let page: Int
    let limit: Int
}

struct KYCResponse: Codable {
    let requests: [KycRequest]
    let total: Int
    let page: Int
}

struct TransactionsResponse: Codable {
    let transactions: [Transaction]
    let total: Int
    let page: Int
}

struct TokensResponse: Codable {
    let tokens: [Token]
}
