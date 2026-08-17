package com.tigeruserwallet.fragments

import android.os.Bundle
import android.provider.Settings
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.ProgressBar
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class StartFragment : Fragment() {

    interface StartHost {
        fun onGuestReady(mode: Mode)
    }

    enum class Mode { CREATE, IMPORT }

    private lateinit var createButton: Button
    private lateinit var importButton: Button
    private lateinit var progressBar: ProgressBar

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_start, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        createButton = view.findViewById(R.id.createWalletButton)
        importButton = view.findViewById(R.id.importWalletButton)
        progressBar = view.findViewById(R.id.startProgressBar)

        createButton.setOnClickListener { bootstrap(Mode.CREATE) }
        importButton.setOnClickListener { bootstrap(Mode.IMPORT) }
    }

    private fun bootstrap(mode: Mode) {
        createButton.isEnabled = false
        importButton.isEnabled = false
        progressBar.visibility = View.VISIBLE
        CoroutineScope(Dispatchers.IO).launch {
            try {
                if (!UserWalletApiService.isAuthenticated()) {
                    UserWalletApiService.guestAuth(deviceId())
                }
                withContext(Dispatchers.Main) {
                    progressBar.visibility = View.GONE
                    createButton.isEnabled = true
                    importButton.isEnabled = true
                    (activity as? StartHost)?.onGuestReady(mode)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    progressBar.visibility = View.GONE
                    createButton.isEnabled = true
                    importButton.isEnabled = true
                    Toast.makeText(
                        requireContext(),
                        e.message ?: "Failed to start guest session",
                        Toast.LENGTH_LONG
                    ).show()
                }
            }
        }
    }

    private fun deviceId(): String {
        val ctx = requireContext()
        val prefs = ctx.getSharedPreferences("userwallet_prefs", android.content.Context.MODE_PRIVATE)
        var id = prefs.getString("device_id", null)
        if (id.isNullOrEmpty()) {
            id = Settings.Secure.getString(ctx.contentResolver, Settings.Secure.ANDROID_ID)
            if (id.isNullOrEmpty()) id = "android-" + System.currentTimeMillis()
            prefs.edit().putString("device_id", id).apply()
        }
        return id
    }
}
