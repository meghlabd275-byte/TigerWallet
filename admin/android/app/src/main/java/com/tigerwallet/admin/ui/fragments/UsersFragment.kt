package com.tigerwallet.admin.ui.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.tigerwallet.admin.R
import com.tigerwallet.admin.data.model.User
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * Users Fragment
 * Display and manage platform users
 */
class UsersFragment : Fragment() {
    
    private lateinit var recyclerView: RecyclerView
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var repository: AdminRepository
    
    private var users = mutableListOf<User>()
    private lateinit var adapter: UsersAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_users, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        recyclerView = view.findViewById(R.id.usersRecyclerView)
        progressIndicator = view.findViewById(R.id.progressIndicator)
        
        repository = AdminRepository(requireContext())
        
        adapter = UsersAdapter(users) { user ->
            showUserActions(user)
        }
        
        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter
        
        loadUsers()
    }
    
    private fun loadUsers() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getUsers()
                if (response.isSuccessful) {
                    users.clear()
                    response.body()?.let { users.addAll(it) }
                    adapter.notifyDataSetChanged()
                } else {
                    Toast.makeText(requireContext(), "Failed to load users", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun showUserActions(user: User) {
        // Show user actions dialog (suspend, verify, view details)
        Toast.makeText(requireContext(), "User: ${user.email}", Toast.LENGTH_SHORT).show()
    }
}

/**
 * Users Adapter
 */
class UsersAdapter(
    private val users: List<User>,
    private val onItemClick: (User) -> Unit
) : RecyclerView.Adapter<UsersAdapter.ViewHolder>() {
    
    class ViewHolder(view: View) : RecyclerView.ViewHolder(view)
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_user, parent, false)
        return ViewHolder(view)
    }
    
    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val user = users[position]
        holder.itemView.setOnClickListener { onItemClick(user) }
    }
    
    override fun getItemCount(): Int = users.size
}
