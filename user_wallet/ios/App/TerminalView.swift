import SwiftUI

// Trading Terminal: live 24h ticker (/terminal/ticker) + OHLC candles
// (/terminal/kline) rendered in a real candle stick chart via Canvas (SwiftUI
// 15+). Data comes from the backend's real CoinGecko-backed endpoints.
struct Candle: Identifiable {
    var id: Int
    var open: Double
    var high: Double
    var low: Double
    var close: Double
}

struct TerminalView: View {
    @State private var symbol = "ETH"
    @State private var days = 1
    @State private var ticker: [String: Any]?
    @State private var candles: [Candle] = []
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationView {
            Form {
                Section("Symbol / Range") {
                    TextField("Symbol", text: $symbol)
                        .autocapitalization(.allCharacters)
                    Picker("Range", selection: $days) {
                        Text("1 day").tag(1)
                        Text("7 days").tag(7)
                        Text("30 days").tag(30)
                    }
                    Button("Load") { load() }
                }
                if let ticker = ticker {
                    Section("24h Ticker") {
                        ForEach(ticker.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                            LabeledContent(key, value: String(describing: value))
                        }
                    }
                }
                Section("Chart") {
                    if candles.isEmpty {
                        Text(isLoading ? "Loading candles…" : "No candle data for this symbol/range")
                            .foregroundColor(.secondary)
                    } else {
                        if #available(iOS 15, *) {
                            CandleChartView(candles: candles)
                                .frame(height: 220)
                        } else {
                            SimpleCandleList(candles: candles)
                        }
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Terminal")
        }
    }

    private func load() {
        isLoading = true
        errorMessage = nil
        let sym = symbol.trimmingCharacters(in: .whitespaces).uppercased()
        let d = days
        Task {
            do {
                let t = try await UserWalletApiService.shared.getTerminalTicker(symbol: sym)
                await MainActor.run { self.ticker = t }
            } catch { }
            do {
                let raw = try await UserWalletApiService.shared.getTerminalKline(symbol: sym, days: d)
                let parsed = TerminalView.parseCandles(raw)
                await MainActor.run { self.candles = parsed; self.isLoading = false }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                    self.candles = []
                }
            }
        }
    }

    /// Parse the backend kline payload (array of arrays or array of objects)
    /// into Candle values. Returns real fetched data only.
    static func parseCandles(_ raw: [String: Any]) -> [Candle] {
        let list: [Any] = (raw["candles"] as? [Any]) ?? (raw["kline"] as? [Any]) ?? (raw["data"] as? [Any]) ?? []
        var out: [Candle] = []
        out.reserveCapacity(list.count)
        for (i, el) in list.enumerated() {
            if let arr = el as? [Any], arr.count >= 5,
               let o = num(arr[1]), let h = num(arr[2]), let l = num(arr[3]), let c = num(arr[4]) {
                out.append(Candle(id: i, open: o, high: h, low: l, close: c))
            } else if let obj = el as? [String: Any],
                      let o = num(obj["open"] ?? obj["o"]),
                      let h = num(obj["high"] ?? obj["h"]),
                      let l = num(obj["low"] ?? obj["l"]),
                      let c = num(obj["close"] ?? obj["c"]) {
                out.append(Candle(id: i, open: o, high: h, low: l, close: c))
            }
        }
        return out
    }

    private static func num(_ v: Any?) -> Double? {
        if let d = v as? Double { return d }
        if let n = v as? NSNumber { return n.doubleValue }
        if let s = v as? String { return Double(s) }
        return nil
    }
}

// MARK: - Canvas candle chart (iOS 15+)

struct CandleChartView: View {
    let candles: [Candle]

    var body: some View {
        Canvas { context, size in
            guard !candles.isEmpty else { return }
            var minV = Double.greatestFiniteMagnitude
            var maxV = -Double.greatestFiniteMagnitude
            for c in candles {
                minV = min(minV, c.low)
                maxV = max(maxV, c.high)
            }
            let span = maxV - minV
            let padX: CGFloat = 8
            let bw = (size.width - padX * 2) / CGFloat(candles.count) - 2
            let safeSpan = span > 0 ? span : 1
            func y(_ v: Double) -> CGFloat {
                size.height - ((v - minV) / safeSpan) * (size.height - 16) - 8
            }
            for (i, c) in candles.enumerated() {
                let up = c.close >= c.open
                let color: Color = up ? .green : .red
                let x = padX + CGFloat(i) * (bw + 2)
                // Wick
                var wick = Path()
                wick.move(to: CGPoint(x: x + bw / 2, y: y(c.high)))
                wick.addLine(to: CGPoint(x: x + bw / 2, y: y(c.low)))
                context.stroke(wick, with: .color(color), lineWidth: 1)
                // Body
                let top = y(max(c.open, c.close))
                let bottom = y(min(c.open, c.close))
                let height = max(bottom - top, 1)
                context.fill(Path(CGRect(x: x, y: top, width: bw, height: height)), with: .color(color))
            }
        }
    }
}

// MARK: - Fallback list (iOS < 15)

struct SimpleCandleList: View {
    let candles: [Candle]
    var body: some View {
        ForEach(candles) { c in
            Text("O \(c.open) H \(c.high) L \(c.low) C \(c.close)")
                .font(.caption.monospaced())
                .foregroundColor(c.close >= c.open ? .green : .red)
        }
    }
}
