package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.tigeruserwallet.MainActivity
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.databinding.FragmentSettingsBinding
import com.tigeruserwallet.ui.ThemeManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Settings (mirrors web Settings.tsx):
 *  - Backend Health (real GET /health)
 *  - Account (the transparent session's user identity)
 *  - Appearance (light/dark via ThemeManager)
 *  - Security -> Reset Wallet (logout + clear local wallet ids -> onboarding)
 *
 * No stubs: health is a real fetch; the theme toggle persists via ThemeManager.
 */
class SettingsFragment : Fragment() {

    private var _binding: FragmentSettingsBinding? = null
    private val binding get() = _binding!!

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentSettingsBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        binding.themeButton.setOnClickListener {
            ThemeManager.toggle()
            requireActivity().recreate()
        }
        binding.logoutButton.setOnClickListener {
            UserWalletApiService.logout()
            (activity as? MainActivity)?.showOnboarding()
        }
        loadHealth()
        loadAccount()
        refreshThemeLabel()
    }

    private fun loadHealth() {
        binding.healthStatus.text = getString(R.string.settings_checking)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val h = UserWalletApiService.health()
                withContext(Dispatchers.Main) {
                    binding.healthStatus.text = h.status.ifEmpty { "unknown" }
                    binding.healthService.text = h.service.ifEmpty { "—" }
                    binding.healthLicensed.text = if (h.licensed)
                        getString(R.string.settings_active) else getString(R.string.settings_inactive)
                    binding.healthClientId.text = h.wlClientId.ifEmpty { "—" }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.healthStatus.text = getString(R.string.settings_health_failed)
                    binding.healthService.text = "—"
                    binding.healthLicensed.text = "—"
                    binding.healthClientId.text = "—"
                }
            }
        }
    }

    private fun loadAccount() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val user = UserWalletApiService.ensureSession().user
                withContext(Dispatchers.Main) {
                    binding.accountEmail.text = user.email.ifEmpty { "—" }
                    binding.accountUsername.text = user.username.ifEmpty { "—" }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.accountEmail.text = "—"
                    binding.accountUsername.text = "—"
                }
            }
        }
    }

    private fun refreshThemeLabel() {
        binding.themeButton.text = getString(
            R.string.settings_theme_current,
            if (ThemeManager.isDark())
                getString(R.string.settings_theme_dark)
            else getString(R.string.settings_theme_light)
        )
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
