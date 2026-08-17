import SwiftUI

// Address Book tab. Lists real saved contacts via getAddressBookContacts and
// supports add / update / delete. The backend returns opaque JSON, so contacts
// are decoded leniently from [String: Any] into a lightweight struct.
struct AddressBookView: View {
    @State private var contacts: [Contact] = []
    @State private var isLoading = false
    @State private var errorMessage: String?

    @State private var showingEditor = false
    @State private var editing: Contact?

    struct Contact: Identifiable {
        let id: String
        var name: String
        var address: String
        var chainId: Int?
    }

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading contacts...")
                } else if let errorMessage = errorMessage {
                    VStack(spacing: 8) {
                        Text(errorMessage).foregroundColor(.red).font(.subheadline)
                            .multilineTextAlignment(.center)
                        Button("Retry", action: loadContacts).buttonStyle(.bordered)
                    }.padding()
                } else if contacts.isEmpty {
                    VStack(spacing: 12) {
                        Text("No contacts yet")
                            .foregroundColor(.secondary)
                        Text("Tap + to add a saved address.")
                            .font(.caption).foregroundColor(.secondary)
                    }
                } else {
                    List {
                        ForEach(contacts) { contact in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(contact.name.isEmpty ? "(unnamed)" : contact.name)
                                    .font(.headline)
                                Text(contact.address)
                                    .font(.caption.monospaced())
                                    .foregroundColor(.secondary)
                                    .textSelection(.enabled)
                                if let chainId = contact.chainId {
                                    Text("Chain #\(chainId)")
                                        .font(.caption2).foregroundColor(.secondary)
                                }
                            }
                            .swipeActions(edge: .trailing) {
                                Button(role: .destructive) { delete(contact) } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                                Button { editing = contact; showingEditor = true } label: {
                                    Label("Edit", systemImage: "pencil")
                                }.tint(.orange)
                            }
                            .contentShape(Rectangle())
                            .onTapGesture {
                                UIPasteboard.general.string = contact.address
                            }
                        }
                    }
                }
            }
            .navigationTitle("Address Book")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        editing = nil
                        showingEditor = true
                    } label: { Image(systemName: "plus") }
                }
            }
            .onAppear { loadContacts() }
            .sheet(isPresented: $showingEditor) {
                ContactEditor(existing: editing) {
                    loadContacts()
                }
            }
        }
    }

    private func loadContacts() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let raw = try await UserWalletApiService.shared.getAddressBookContacts()
                let parsed = Self.parseContacts(raw)
                await MainActor.run {
                    self.contacts = parsed
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

    // The backend response shape is { contacts: [...] } or a bare array.
    static func parseContacts(_ raw: [String: Any]) -> [Contact] {
        let arr: [[String: Any]]
        if let list = raw["contacts"] as? [[String: Any]] {
            arr = list
        } else if let list = raw["data"] as? [[String: Any]] {
            arr = list
        } else {
            arr = []
        }
        return arr.compactMap { item in
            let id = (item["id"] as? String) ?? (item["uuid"] as? String) ?? UUID().uuidString
            return Contact(
                id: id,
                name: (item["name"] as? String) ?? "",
                address: (item["address"] as? String) ?? "",
                chainId: (item["chain_id"] as? Int) ?? (item["chainId"] as? Int))
        }
    }

    private func delete(_ contact: Contact) {
        Task {
            do {
                _ = try await UserWalletApiService.shared.deleteContact(id: contact.id)
                await MainActor.run {
                    self.contacts.removeAll { $0.id == contact.id }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}

// Add / edit contact sheet.
struct ContactEditor: View {
    let existing: AddressBookView.Contact?
    let onSaved: () -> Void

    @State private var name = ""
    @State private var address = ""
    @State private var chainId: Int = 1
    @State private var error: String?
    @State private var isSaving = false
    @Environment(\.dismiss) var dismiss

    private var isEdit: Bool { existing != nil }

    private var canSave: Bool {
        !isSaving
            && !name.trimmingCharacters(in: .whitespaces).isEmpty
            && !address.trimmingCharacters(in: .whitespaces).isEmpty
    }

    var body: some View {
        NavigationView {
            Form {
                Section("Contact") {
                    TextField("Name", text: $name)
                    TextField("Address (0x...)", text: $address)
                        .autocapitalization(.none).disableAutocorrection(true)
                        .font(.system(.body, design: .monospaced))
                    Picker("Chain", selection: $chainId) {
                        Text("Ethereum").tag(1)
                        Text("BNB Chain").tag(56)
                        Text("Polygon").tag(137)
                    }
                }
                if let error = error {
                    Section { Text(error).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle(isEdit ? "Edit Contact" : "New Contact")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(action: save) {
                        if isSaving {
                            ProgressView().tint(.orange)
                        } else {
                            Text("Save").fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSave)
                }
            }
            .onAppear {
                if let c = existing {
                    name = c.name
                    address = c.address
                    chainId = c.chainId ?? 1
                }
            }
        }
    }

    private func save() {
        isSaving = true
        error = nil
        let n = name.trimmingCharacters(in: .whitespacesAndNewlines)
        let a = address.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                if let existing = existing {
                    _ = try await UserWalletApiService.shared.updateContact(
                        id: existing.id, name: n, address: a, chainId: chainId)
                } else {
                    _ = try await UserWalletApiService.shared.addContact(
                        name: n, address: a, chainId: chainId)
                }
                await MainActor.run {
                    self.isSaving = false
                    self.onSaved()
                    self.dismiss()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isSaving = false
                }
            }
        }
    }
}
