import Foundation

/// UserWallet WalletConnect live-event socket (iOS).
///
/// Connects to the canonical dapp_browser WalletConnect relay through the
/// wallet_api reverse proxy:  ws(s)://<host>/api/v1/dapp/ws/<topic>
///
/// The wire protocol is JSON-RPC-style frames: { id, method, params }.
/// Server-pushed events arrive with a `method` field; client requests elicit
/// responses keyed by `id`. This helper only transports REAL frames — it never
/// fabricates events.
///
/// Usage:
///   let socket = WalletConnectSocket()
///   try socket.connect(topic: topic) { frame in /* [String: Any] */ }
///   try socket.sendRequest(method: "wc_sessionRequest", params: [:])
///   socket.close()
@available(iOS 13.0, *)
final class WalletConnectSocket {

    enum SocketError: Error {
        case invalidURL
        case notConnected
    }

    /// HTTP API base; mirrors UserWalletApiService's default baseURL.
    private static let apiBaseURL = "http://localhost:8443/api/v1"

    static func wsBase() -> String {
        let httpBase = apiBaseURL.hasSuffix("/") ? String(apiBaseURL.dropLast()) : apiBaseURL
        var wsBase = httpBase
        if wsBase.hasPrefix("https") {
            wsBase = "wss" + wsBase.dropFirst("https".count)
        } else if wsBase.hasPrefix("http") {
            wsBase = "ws" + wsBase.dropFirst("http".count)
        }
        return wsBase + "/dapp/ws"
    }

    private let session: URLSession
    private var task: URLSessionWebSocketTask?
    private var nextId: Int64 = 1
    private var isClosed = false

    init(session: URLSession = .shared) {
        self.session = session
    }

    /// Open a live WalletConnect socket for a pairing topic.
    /// - Parameters:
    ///   - topic: the WalletConnect pairing topic
    ///   - onMessage: invoked for every parsed JSON frame (main-queue hop is the caller's job)
    ///   - onFailure: invoked once when the socket fails or closes abnormally
    func connect(topic: String,
                 onMessage: @escaping ([String: Any]) -> Void,
                 onFailure: ((Error) -> Void)? = nil) throws {
        let encoded = topic.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? topic
        guard let url = URL(string: "\(WalletConnectSocket.wsBase())/\(encoded)") else {
            throw SocketError.invalidURL
        }
        let task = session.webSocketTask(with: url)
        self.task = task
        task.resume()
        receiveLoop(onMessage: onMessage, onFailure: onFailure)
    }

    private func receiveLoop(onMessage: @escaping ([String: Any]) -> Void,
                             onFailure: ((Error) -> Void)?) {
        task?.receive { [weak self] result in
            guard let self = self, !self.isClosed else { return }
            switch result {
            case .success(let message):
                var text: String?
                if case .string(let s) = message { text = s }
                if case .data(let d) = message { text = String(data: d, encoding: .utf8) }
                if let text = text,
                   let data = text.data(using: .utf8),
                   let frame = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                    onMessage(frame)
                }
                self.receiveLoop(onMessage: onMessage, onFailure: onFailure)
            case .failure(let error):
                if !self.isClosed {
                    onFailure?(error)
                }
            }
        }
    }

    /// Send a JSON-RPC request frame. Returns the frame id.
    @discardableResult
    func sendRequest(method: String, params: [String: Any]? = nil) async throws -> Int64 {
        guard let task = self.task, !isClosed else { throw SocketError.notConnected }
        let id = nextId
        nextId += 1
        var frame: [String: Any] = ["id": id, "method": method]
        if let params = params { frame["params"] = params }
        let data = try JSONSerialization.data(withJSONObject: frame)
        guard let text = String(data: data, encoding: .utf8) else {
            throw SocketError.notConnected
        }
        try await task.send(.string(text))
        return id
    }

    func close(code: URLSessionWebSocketTask.CloseCode = .normalClosure,
               reason: Data? = nil) {
        isClosed = true
        task?.cancel(with: code, reason: reason)
        task = nil
    }
}
