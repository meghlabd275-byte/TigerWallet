import Foundation

/// UserWallet live price-feed socket (iOS).
///
/// Connects to the canonical wallet_api public live feed:
///   ws(s)://<host>/api/v1/ws
///
/// Protocol: client sends { action: "subscribe", symbols: [...] } and the
/// server pushes { type: "ticker", symbol, last_price, volume_24h,
/// market_cap, change_24h_pct } frames sourced from the live price oracle.
/// This helper only transports REAL frames — it never fabricates tickers.
final class LiveFeedSocket: NSObject, URLSessionWebSocketDelegate {
    private var task: URLSessionWebSocketTask?
    private var session: URLSession?

    var onTicker: (([String: Any]) -> Void)?
    var onClosed: (() -> Void)?

    static func wsURL() -> URL? {
        var wsBase = UserWalletApiService.shared.baseURL
        if wsBase.hasPrefix("https") {
            wsBase = "wss" + wsBase.dropFirst("https".count)
        } else if wsBase.hasPrefix("http") {
            wsBase = "ws" + wsBase.dropFirst("http".count)
        }
        if wsBase.hasSuffix("/") { wsBase.removeLast() }
        return URL(string: wsBase + "/ws")
    }

    func connect(symbols: [String]) {
        guard let url = LiveFeedSocket.wsURL() else { return }
        let session = URLSession(configuration: .default, delegate: self, delegateQueue: nil)
        self.session = session
        let task = session.webSocketTask(with: url)
        self.task = task
        task.resume()
        let sub: [String: Any] = ["action": "subscribe", "symbols": symbols]
        if let data = try? JSONSerialization.data(withJSONObject: sub),
           let text = String(data: data, encoding: .utf8) {
            task.send(.string(text)) { _ in }
        }
        receive()
    }

    private func receive() {
        task?.receive { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .success(let message):
                if case .string(let text) = message,
                   let data = text.data(using: .utf8),
                   let frame = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                   (frame["type"] as? String) == "ticker" {
                    self.onTicker?(frame)
                }
                self.receive()
            case .failure:
                self.onClosed?()
            }
        }
    }

    func close() {
        task?.cancel(with: .normalClosure, reason: nil)
        task = nil
        session?.invalidateAndCancel()
        session = nil
    }
}
