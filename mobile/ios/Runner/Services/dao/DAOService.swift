import Foundation

class DAOService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getDAOs(completion: @escaping ([DAO]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/dao/list") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let daos = array.compactMap { dict -> DAO? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return DAO(id: id, name: name, description: dict["description"] as? String ?? "", memberCount: dict["memberCount"] as? Int ?? 0)
            }
            completion(daos)
        }.resume()
    }
    
    func vote(proposalId: String, choice: String, weight: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/dao/proposals/\(proposalId)/vote") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["choice": choice, "weight": weight])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 200) }
            else { completion(false) }
        }.resume()
    }
}

struct DAO { let id: String; let name: String; let description: String; let memberCount: Int }

class LaunchpadService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getActiveLaunches(completion: @escaping ([Launch]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/launchpad/active") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let launches = array.compactMap { dict -> Launch? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return Launch(id: id, name: name, price: dict["price"] as? Double ?? 0, hardCap: dict["hardCap"] as? Double ?? 0, raised: dict["raised"] as? Double ?? 0)
            }
            completion(launches)
        }.resume()
    }
    
    func participate(launchId: String, amount: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/launchpad/\(launchId)/participate") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["amount": amount])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct Launch { let id: String; let name: String; let price: Double; let hardCap: Double; let raised: Double }

class PredictionMarketsService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getMarkets(completion: @escaping ([PredictionMarket]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/prediction/markets") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let markets = array.compactMap { dict -> PredictionMarket? in
                guard let id = dict["id"] as? String, let question = dict["question"] as? String else { return nil }
                return PredictionMarket(id: id, question: question, volume: dict["volume"] as? Double ?? 0, status: dict["status"] as? String ?? "")
            }
            completion(markets)
        }.resume()
    }
    
    func placeBet(marketId: String, outcome: String, amount: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/prediction/\(marketId)/bet") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["outcome": outcome, "amount": amount])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct PredictionMarket { let id: String; let question: String; let volume: Double; let status: String }

class RWATradingService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getRWAs(completion: @escaping ([RWA]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/rwa/list") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let rwas = array.compactMap { dict -> RWA? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return RWA(id: id, name: name, price: dict["price"] as? Double ?? 0, marketCap: dict["marketCap"] as? Double ?? 0)
            }
            completion(rwas)
        }.resume()
    }
    
    func buyRWA(rwaId: String, amount: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/rwa/\(rwaId)/buy") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["amount": amount])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct RWA { let id: String; let name: String; let price: Double; let marketCap: Double }

class GasTrackerService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    
    func getGasPrice(chain: String, completion: @escaping (GasPrice?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/gas/price?chain=\(chain)") else { return }
        
        URLSession.shared.dataTask(with: url) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let dataObj = json["data"] as? [String: Any] else {
                completion(nil)
                return
            }
            completion(GasPrice(slow: dataObj["slow"] as? Double ?? 0, standard: dataObj["standard"] as? Double ?? 0, fast: dataObj["fast"] as? Double ?? 0))
        }.resume()
    }
}

struct GasPrice { let slow: Double; let standard: Double; let fast: Double }

class OrderbookService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    func getOrderbook(symbol: String, completion: @escaping (Orderbook?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/orderbook/\(symbol)") else { return }
        
        URLSession.shared.dataTask(with: url) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let dataObj = json["data"] as? [String: Any] else {
                completion(nil)
                return
            }
            completion(Orderbook(symbol: dataObj["symbol"] as? String ?? ""))
        }.resume()
    }
}

struct Orderbook { let symbol: String }

class TWAPService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func createTWAP(symbol: String, totalAmount: Double, intervals: Int, side: String, completion: @escaping (String?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/twap/create") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["symbol": symbol, "totalAmount": totalAmount, "intervals": intervals, "side": side])
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let orderId = (json["data"] as? [String: Any])?["id"] as? String else {
                completion(nil)
                return
            }
            completion(orderId)
        }.resume()
    }
}
