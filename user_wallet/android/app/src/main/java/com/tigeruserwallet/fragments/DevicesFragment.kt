package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.ProgressBar
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.DevicesAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Devices: lists paired devices via [getDevices] and exposes register/sync/
 * delete against the backend.
 */
class DevicesFragment : Fragment() {
    private lateinit var nameInput: EditText
    private lateinit var typeInput: EditText
    private lateinit var registerButton: Button
    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView
    private lateinit var recyclerView: RecyclerView

    private val adapter = DevicesAdapter(
        mutableListOf(),
        onSync = { syncDevice(it) },
        onDelete = { deleteDevice(it) }
    )

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_devices, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        nameInput = view.findViewById(R.id.deviceNameInput)
        typeInput = view.findViewById(R.id.deviceTypeInput)
        registerButton = view.findViewById(R.id.deviceRegisterButton)
        progressBar = view.findViewById(R.id.devicesProgressBar)
        statusTextView = view.findViewById(R.id.devicesStatusTextView)
        recyclerView = view.findViewById(R.id.devicesRecyclerView)

        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter

        registerButton.setOnClickListener { registerDevice() }
        loadDevices()
    }

    private fun loadDevices() {
        setLoading(true)
        statusTextView.text = "Loading devices..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val devices = UserWalletApiService.getDevices()
                withContext(Dispatchers.Main) {
                    adapter.update(devices)
                    statusTextView.text =
                        if (devices.isEmpty()) "No paired devices" else "${devices.size} devices"
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load devices"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun registerDevice() {
        val name = nameInput.text.toString().trim()
        val type = typeInput.text.toString().trim()
        if (name.isEmpty() || type.isEmpty()) {
            Toast.makeText(requireContext(), "Enter name and type", Toast.LENGTH_SHORT).show()
            return
        }
        setButtonsEnabled(false)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.registerDevice(name, type)
                withContext(Dispatchers.Main) {
                    nameInput.text.clear()
                    typeInput.text.clear()
                    loadDevices()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Register failed"}"
                }
            } finally {
                withContext(Dispatchers.Main) { setButtonsEnabled(true) }
            }
        }
    }

    private fun syncDevice(device: JSONObject) {
        val id = device.optString("id", device.optString("device_id", ""))
        if (id.isEmpty()) return
        setLoading(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.syncDevice(id)
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "\u2713 Device sync requested", Toast.LENGTH_SHORT).show()
                    loadDevices()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Sync failed"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun deleteDevice(device: JSONObject) {
        val id = device.optString("id", device.optString("device_id", ""))
        if (id.isEmpty()) return
        setLoading(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.deleteDevice(id)
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "\u2713 Device removed", Toast.LENGTH_SHORT).show()
                    loadDevices()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Delete failed"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun setButtonsEnabled(enabled: Boolean) {
        registerButton.isEnabled = enabled
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
