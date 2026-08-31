import SwiftUI

// DAO governance: proposals list + create, voting, delegates.
// GET/POST /dao/proposals, POST /dao/proposals/:id/vote, GET /dao/delegates.
struct DAOView: View {
    @State private var proposals: [[String: Any]] = []
    @State private var delegates: [[String: Any]] = []
    @State private var title = ""
    @State private var proposalDescription = ""
    @State private var voteProposalId = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    var body: some View {
        NavigationView {
            Form {
                Section("Proposals") {
                    if proposals.isEmpty {
                        Text(isLoading ? "Loading…" : "No active proposals")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(proposals.enumerated()), id: \.offset) { _, p in
                            let id = (p["id"] ?? "") as Any
                            let t = (p["title"] ?? "?") as Any
                            let yes = (p["votes_for"] ?? p["yes"] ?? 0) as Any
                            let no = (p["votes_against"] ?? p["no"] ?? 0) as Any
                            Text("• \(String(describing: id)): \(String(describing: t)) — yes:\(String(describing: yes)) no:\(String(describing: no))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Create Proposal") {
                    TextField("Title", text: $title)
                    TextField("Description", text: $proposalDescription)
                    Button("Create Proposal") { create() }
                        .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty
                                  || proposalDescription.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                Section("Vote") {
                    TextField("Proposal ID", text: $voteProposalId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    HStack {
                        Button("Vote Yes") { vote(true) }
                            .disabled(voteProposalId.trimmingCharacters(in: .whitespaces).isEmpty)
                        Button("Vote No") { vote(false) }
                            .disabled(voteProposalId.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                }
                Section("Delegates") {
                    if delegates.isEmpty {
                        Text("No delegates").foregroundColor(.secondary)
                    } else {
                        ForEach(Array(delegates.enumerated()), id: \.offset) { _, d in
                            let addr = (d["address"] ?? d["name"] ?? "?") as Any
                            let power = (d["voting_power"] ?? d["power"] ?? "?") as Any
                            Text("• \(String(describing: addr)) — \(String(describing: power))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("DAO Governance")
            .onAppear { load() }
            .alert(isPresented: $showSuccess) {
                Alert(title: Text("\u{2713} Submitted"),
                      message: Text(successDetail),
                      dismissButton: .default(Text("OK")))
            }
        }
    }

    private func load() {
        isLoading = true
        Task {
            do {
                let pRes = try await UserWalletApiService.shared.getDaoProposals()
                let dRes = try await UserWalletApiService.shared.getDaoDelegates()
                let pList = (pRes["data"] as? [[String: Any]]) ?? (pRes["proposals"] as? [[String: Any]]) ?? []
                let dList = (dRes["delegates"] as? [[String: Any]]) ?? (dRes["data"] as? [[String: Any]]) ?? []
                await MainActor.run {
                    self.proposals = pList
                    self.delegates = dList
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

    private func create() {
        errorMessage = nil
        Task {
            do {
                let res = try await UserWalletApiService.shared.createDaoProposal(
                    title: title.trimmingCharacters(in: .whitespaces),
                    description: proposalDescription.trimmingCharacters(in: .whitespaces))
                await MainActor.run {
                    self.successDetail = "Proposal created: \(String(describing: res["id"] ?? res["proposal_id"] ?? "ok"))"
                    self.showSuccess = true
                    self.title = ""
                    self.proposalDescription = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func vote(_ support: Bool) {
        errorMessage = nil
        let id = voteProposalId.trimmingCharacters(in: .whitespaces)
        Task {
            do {
                _ = try await UserWalletApiService.shared.voteDaoProposal(proposalId: id, support: support)
                await MainActor.run {
                    self.successDetail = "Vote submitted to the blockchain network"
                    self.showSuccess = true
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
