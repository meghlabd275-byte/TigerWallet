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
import com.tigeruserwallet.adapters.ContactsAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Address book: lists real contacts from [getAddressBookContacts] and exposes
 * add/update/delete against the backend. Selecting a row is not required for
 * delete (the row's own Delete button carries the contact id); Update applies
 * the current Name/Address fields to the most recently selected contact.
 */
class AddressBookFragment : Fragment() {
    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView
    private lateinit var recyclerView: RecyclerView
    private lateinit var nameInput: EditText
    private lateinit var addressInput: EditText
    private lateinit var addButton: Button
    private lateinit var updateButton: Button

    private val adapter = ContactsAdapter(mutableListOf()) { deleteContact(it) }
    private var selectedId: String? = null

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_address_book, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        progressBar = view.findViewById(R.id.addressBookProgressBar)
        statusTextView = view.findViewById(R.id.addressBookStatusTextView)
        recyclerView = view.findViewById(R.id.addressBookRecyclerView)
        nameInput = view.findViewById(R.id.abNameInput)
        addressInput = view.findViewById(R.id.abAddressInput)
        addButton = view.findViewById(R.id.abAddButton)
        updateButton = view.findViewById(R.id.abUpdateButton)

        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter

        adapter.setOnItemClickListener { contact ->
            selectedId = contact.optString("id")
            nameInput.setText(contact.optString("name", ""))
            addressInput.setText(contact.optString("address", ""))
        }

        addButton.setOnClickListener { addContact() }
        updateButton.setOnClickListener { updateContact() }
        loadContacts()
    }

    private fun loadContacts() {
        setLoading(true)
        statusTextView.text = "Loading contacts..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val contacts = UserWalletApiService.getAddressBookContacts()
                withContext(Dispatchers.Main) {
                    adapter.update(contacts)
                    statusTextView.text =
                        if (contacts.isEmpty()) "No contacts yet" else "${contacts.size} contacts"
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load contacts"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun addContact() {
        val name = nameInput.text.toString().trim()
        val address = addressInput.text.toString().trim()
        if (name.isEmpty() || address.isEmpty()) {
            Toast.makeText(requireContext(), "Enter name and address", Toast.LENGTH_SHORT).show()
            return
        }
        setButtonsEnabled(false)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.addContact(name, address, null)
                withContext(Dispatchers.Main) {
                    nameInput.text.clear()
                    addressInput.text.clear()
                    loadContacts()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Add failed"}"
                }
            } finally {
                withContext(Dispatchers.Main) { setButtonsEnabled(true) }
            }
        }
    }

    private fun updateContact() {
        val id = selectedId ?: run {
            Toast.makeText(requireContext(), "Select a contact first", Toast.LENGTH_SHORT).show()
            return
        }
        val name = nameInput.text.toString().trim()
        val address = addressInput.text.toString().trim()
        if (name.isEmpty() && address.isEmpty()) {
            Toast.makeText(requireContext(), "Enter a name or address to update", Toast.LENGTH_SHORT).show()
            return
        }
        setButtonsEnabled(false)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.updateContact(id,
                    name.ifEmpty { null },
                    address.ifEmpty { null },
                    null)
                withContext(Dispatchers.Main) { loadContacts() }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Update failed"}"
                }
            } finally {
                withContext(Dispatchers.Main) { setButtonsEnabled(true) }
            }
        }
    }

    private fun deleteContact(contact: JSONObject) {
        val id = contact.optString("id")
        if (id.isEmpty()) return
        setButtonsEnabled(false)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.deleteContact(id)
                withContext(Dispatchers.Main) {
                    if (selectedId == id) {
                        selectedId = null
                        nameInput.text.clear()
                        addressInput.text.clear()
                    }
                    loadContacts()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Delete failed"}"
                }
            } finally {
                withContext(Dispatchers.Main) { setButtonsEnabled(true) }
            }
        }
    }

    private fun setButtonsEnabled(enabled: Boolean) {
        addButton.isEnabled = enabled
        updateButton.isEnabled = enabled
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
