package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.tigeruserwallet.R

/**
 * Onboarding (no-registration self-custody) — mirrors web OnboardingChoice.
 *
 * The app opens here when no local wallet exists. There is NO login/register
 * form; a transparent ephemeral session is auto-provisioned by the first
 * wallet-create/import call (UserWalletApiService.ensureSession). The user
 * only chooses between Create (new seed) and Import (existing seed).
 */
class OnboardingFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        return inflater.inflate(R.layout.fragment_onboarding, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        view.findViewById<MaterialButton>(R.id.btnCreateWallet).setOnClickListener {
            parentFragmentManager.beginTransaction()
                .replace(R.id.fragmentContainer, CreateWalletFragment())
                .addToBackStack(null)
                .commit()
        }
        view.findViewById<MaterialButton>(R.id.btnImportWallet).setOnClickListener {
            parentFragmentManager.beginTransaction()
                .replace(R.id.fragmentContainer, ImportWalletFragment())
                .addToBackStack(null)
                .commit()
        }
    }
}
