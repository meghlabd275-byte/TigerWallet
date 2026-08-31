package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * DAO governance: proposals list + create, voting, delegates.
 * GET/POST /dao/proposals, POST /dao/proposals/:id/vote, GET /dao/delegates.
 */
class DaoFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_dao, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val proposalsText = view.findViewById<TextView>(R.id.daoProposalsText)
        val delegatesText = view.findViewById<TextView>(R.id.daoDelegatesText)
        val statusText = view.findViewById<TextView>(R.id.daoStatusText)

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val proposals = UserWalletApiService.getDaoProposals()
                    val delegates = UserWalletApiService.getDaoDelegates()
                    val delegateArr = delegates.optJSONArray("delegates")
                        ?: delegates.optJSONArray("data")
                    withContext(Dispatchers.Main) {
                        proposalsText.text = if (proposals.isEmpty()) "No active proposals"
                        else proposals.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("title", "?")} — yes:${it.optInt("votes_for", it.optInt("yes", 0))} no:${it.optInt("votes_against", it.optInt("no", 0))}"
                        }
                        delegatesText.text = if (delegateArr == null || delegateArr.length() == 0) "No delegates"
                        else (0 until delegateArr.length()).joinToString("\n") { i ->
                            val d = delegateArr.getJSONObject(i)
                            "\u2022 ${d.optString("address", d.optString("name", "?"))} — ${d.optString("voting_power", d.optString("power", "?"))}"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { proposalsText.text = "DAO data unavailable" }
                }
            }
        }
        load()

        view.findViewById<Button>(R.id.daoCreateButton).setOnClickListener {
            val title = view.findViewById<EditText>(R.id.daoTitleInput).text.toString().trim()
            val desc = view.findViewById<EditText>(R.id.daoDescriptionInput).text.toString().trim()
            if (title.isEmpty() || desc.isEmpty()) {
                Toast.makeText(requireContext(), "Enter title and description", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Submitting proposal…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.createDaoProposal(title, desc)
                    withContext(Dispatchers.Main) {
                        statusText.text = "Proposal created: ${res.optString("id", res.optString("proposal_id", "ok"))}"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Create failed: ${e.message}" }
                }
            }
        }

        fun vote(support: Boolean) {
            val id = view.findViewById<EditText>(R.id.daoProposalIdInput).text.toString().trim()
            if (id.isEmpty()) {
                Toast.makeText(requireContext(), "Enter proposal ID", Toast.LENGTH_SHORT).show()
                return
            }
            statusText.text = "Voting…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.voteDaoProposal(id, support)
                    withContext(Dispatchers.Main) {
                        statusText.text = "Vote submitted to the blockchain network"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Vote failed: ${e.message}" }
                }
            }
        }
        view.findViewById<Button>(R.id.daoVoteYesButton).setOnClickListener { vote(true) }
        view.findViewById<Button>(R.id.daoVoteNoButton).setOnClickListener { vote(false) }
    }
}
