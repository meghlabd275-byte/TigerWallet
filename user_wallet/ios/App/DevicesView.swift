import SwiftUI

// Devices tab. Lists real registered devices via getDevices and supports
// register / sync / delete. Mirrors the web /devices page.
struct DevicesView: View {
    @State private var devices: [Device] = []
    @State private var isLoading = false
    @State private var errorMessage: String?

    @State private var showingRegister = false
    @State private var syncingId: String?

    struct Device: Identifiable {
        let id: String
        let name: String
        let type: String
        let lastSeen: String?
    }

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading devices...")
                } else if let errorMessage = errorMessage {
                    VStack(spacing: 8) {
                        Text(errorMessage).foregroundColor(.red).font(.subheadline)
                            .multilineTextAlignment(.center)
                        Button("Retry", action: loadDevices).buttonStyle(.bordered)
                    }.padding()
                } else if devices.isEmpty {
                    VStack(spacing: 12) {
                        Text("No devices registered")
                            .foregroundColor(.secondary)
                        Text("Tap + to register this device.")
                            .font(.caption).foregroundColor(.secondary)
                    }
                } else {
                    List {
                        ForEach(devices) { device in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(device.name.isEmpty ? device.id : device.name)
                                    .font(.headline)
                                Text("Type: \(device.type.isEmpty ? "unknown" : device.type)")
                                    .font(.caption).foregroundColor(.secondary)
                                if let lastSeen = device.lastSeen, !lastSeen.isEmpty {
                                    Text("Last seen: \(lastSeen)")
                                        .font(.caption2).foregroundColor(.secondary)
                                }
                            }
                            .swipeActions(edge: .trailing) {
                                Button(role: .destructive) { delete(device) } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                                Button { sync(device) } label: {
                                    Label("Sync", systemImage: "arrow.triangle.2.circlepath")
                                }.tint(.orange)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Devices")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { showingRegister = true } label: { Image(systemName: "plus") }
                }
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: loadDevices) { Image(systemName: "arrow.clockwise") }
                }
            }
            .onAppear { loadDevices() }
            .sheet(isPresented: $showingRegister) {
                DeviceRegisterView { loadDevices() }
            }
        }
    }

    private func loadDevices() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let raw = try await UserWalletApiService.shared.getDevices()
                let parsed = Self.parseDevices(raw)
                await MainActor.run {
                    self.devices = parsed
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    static func parseDevices(_ raw: [String: Any]) -> [Device] {
        let arr: [[String: Any]]
        if let list = raw["devices"] as? [[String: Any]] {
            arr = list
        } else if let list = raw["data"] as? [[String: Any]] {
            arr = list
        } else {
            arr = []
        }
        return arr.enumerated().map { idx, item in
            Device(
                id: (item["id"] as? String) ?? (item["device_id"] as? String) ?? "\(idx)",
                name: (item["name"] as? String) ?? "",
                type: (item["device_type"] as? String) ?? (item["type"] as? String) ?? "",
                lastSeen: (item["last_seen"] as? String) ?? (item["lastSync"] as? String))
        }
    }

    private func sync(_ device: Device) {
        syncingId = device.id
        errorMessage = nil
        Task {
            do {
                _ = try await UserWalletApiService.shared.syncDevice(deviceId: device.id)
                await MainActor.run { self.syncingId = nil; self.loadDevices() }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.syncingId = nil
                }
            }
        }
    }

    private func delete(_ device: Device) {
        Task {
            do {
                _ = try await UserWalletApiService.shared.deleteDevice(deviceId: device.id)
                await MainActor.run {
                    self.devices.removeAll { $0.id == device.id }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}

struct DeviceRegisterView: View {
    let onRegistered: () -> Void

    @State private var name = ""
    @State private var deviceType = "ios"
    @State private var error: String?
    @State private var isRegistering = false
    @Environment(\.dismiss) var dismiss

    private var canRegister: Bool {
        !isRegistering && !name.trimmingCharacters(in: .whitespaces).isEmpty
    }

    var body: some View {
        NavigationView {
            Form {
                Section("Device") {
                    TextField("Name", text: $name)
                    Picker("Type", selection: $deviceType) {
                        Text("iOS").tag("ios")
                        Text("macOS").tag("macos")
                        Text("Web").tag("web")
                        Text("Android").tag("android")
                    }
                }
                if let error = error {
                    Section { Text(error).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Register Device")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(action: register) {
                        if isRegistering {
                            ProgressView().tint(.orange)
                        } else {
                            Text("Register").fontWeight(.semibold)
                        }
                    }
                    .disabled(!canRegister)
                }
            }
        }
    }

    private func register() {
        isRegistering = true
        error = nil
        let n = name.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                _ = try await UserWalletApiService.shared.registerDevice(
                    name: n, deviceType: deviceType)
                await MainActor.run {
                    self.isRegistering = false
                    self.onRegistered()
                    self.dismiss()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isRegistering = false
                }
            }
        }
    }
}
