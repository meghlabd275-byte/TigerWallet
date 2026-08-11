import Foundation

/**
 * Gift Card Service - iOS Native Implementation
 */
class GiftCardService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String? = nil) {
        self.authToken = token
    }
    
    private func createRequest(endpoint: String, method: String = "GET", body: [String: Any]? = nil) -> URLRequest? {
        guard let url = URL(string: "\(Self.API_BASE)\(endpoint)") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if let body = body {
            request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        }
        return request
    }
    
    /// Get templates
    func getTemplates(completion: @escaping (Result<[GiftCardTemplate], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/giftcards/templates") else {
            completion(.failure(GiftCardError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let templatesData = json["data"] as? [[String: Any]] else {
                completion(.failure(GiftCardError.invalidResponse)); return
            }
            
            let templates = templatesData.compactMap { t -> GiftCardTemplate? in
                guard let id = t["id"] as? String else { return nil }
                return GiftCardTemplate(id: id, name: t["name"] as? String ?? "", imageUrl: t["imageUrl"] as? String ?? "", isActive: t["isActive"] as? Bool ?? false)
            }
            completion(.success(templates))
        }.resume()
    }
    
    /// Create gift card
    func createGiftCard(token: String, amount: Double, templateId: String?, completion: @escaping (Result<GiftCard, Error>) -> Void) {
        var body: [String: Any] = ["token": token, "amount": amount]
        if let templateId = templateId { body["templateId"] = templateId }
        
        guard let request = createRequest(endpoint: "/giftcards", method: "POST", body: body) else {
            completion(.failure(GiftCardError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let cardData = json["data"] as? [String: Any] else {
                completion(.failure(GiftCardError.invalidResponse)); return
            }
            
            let card = GiftCard(
                id: cardData["id"] as? String ?? "",
                code: cardData["code"] as? String ?? "",
                token: cardData["token"] as? String ?? "",
                amount: cardData["amount"] as? Double ?? 0,
                status: cardData["status"] as? String ?? "ACTIVE"
            )
            completion(.success(card))
        }.resume()
    }
    
    /// Redeem gift card
    func redeemGiftCard(code: String, completion: @escaping (Result<GiftCard, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/giftcards/redeem", method: "POST", body: ["code": code]) else {
            completion(.failure(GiftCardError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let cardData = json["data"] as? [String: Any] else {
                completion(.failure(GiftCardError.invalidResponse)); return
            }
            
            let card = GiftCard(
                id: cardData["id"] as? String ?? "",
                code: cardData["code"] as? String ?? "",
                token: cardData["token"] as? String ?? "",
                amount: cardData["amount"] as? Double ?? 0,
                status: cardData["status"] as? String ?? "REDEEMED"
            )
            completion(.success(card))
        }.resume()
    }
    
    /// Check balance
    func checkBalance(code: String, completion: @escaping (Result<GiftCard?, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/giftcards/\(code)/balance") else {
            completion(.failure(GiftCardError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error { completion(.failure(error)); return }
            guard let httpResponse = response as? HTTPURLResponse else {
                completion(.failure(GiftCardError.invalidResponse)); return
            }
            
            if httpResponse.statusCode == 404 {
                completion(.success(nil)); return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let cardData = json["data"] as? [String: Any] else {
                completion(.failure(GiftCardError.invalidResponse)); return
            }
            
            let card = GiftCard(
                id: cardData["id"] as? String ?? "",
                code: cardData["code"] as? String ?? "",
                token: cardData["token"] as? String ?? "",
                amount: cardData["amount"] as? Double ?? 0,
                status: cardData["status"] as? String ?? "ACTIVE"
            )
            completion(.success(card))
        }.resume()
    }
}

struct GiftCardTemplate {
    let id: String
    let name: String
    let imageUrl: String
    let isActive: Bool
}

struct GiftCard {
    let id: String
    let code: String
    let token: String
    let amount: Double
    let status: String
}

enum GiftCardError: Error {
    case invalidRequest
    case invalidResponse
}
