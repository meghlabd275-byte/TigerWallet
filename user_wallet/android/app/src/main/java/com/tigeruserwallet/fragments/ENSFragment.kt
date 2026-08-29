package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * ENS: forward resolution (GET /ens/resolve) and reverse lookup
 * (GET /ens/lookup) against the real on-chain ENS registry via the backend.
 */
class ENSFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_ens, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val nameInput = view.findViewById<EditText>(R.id.ensNameInput)
        val resolveResult = view.findViewById<TextView>(R.id.ensResolveResult)
        val addressInput = view.findViewById<EditText>(R.id.ensAddressInput)
        val lookupResult = view.findViewById<TextView>(R.id.ensLookupResult)

        view.findViewById<Button>(R.id.ensResolveButton).setOnClickListener {
            val name = nameInput.text.toString().trim()
            if (name.isEmpty()) { resolveResult.text = "Enter an ENS name"; return@setOnClickListener }
            resolveResult.text = "Resolving…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.resolveENS(name)
                    withContext(Dispatchers.Main) {
                        resolveResult.text = if (res.address.isNotEmpty()) "$name → ${res.address}" else "No address found"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { resolveResult.text = "Resolution failed: ${e.message}" }
                }
            }
        }

        view.findViewById<Button>(R.id.ensLookupButton).setOnClickListener {
            val address = addressInput.text.toString().trim()
            if (address.isEmpty()) { lookupResult.text = "Enter an address"; return@setOnClickListener }
            lookupResult.text = "Looking up…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.lookupENS(address)
                    withContext(Dispatchers.Main) {
                        lookupResult.text = if (res.name.isNotEmpty()) "$address → ${res.name}" else "No name found"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { lookupResult.text = "Lookup failed: ${e.message}" }
                }
            }
        }
    }
}
