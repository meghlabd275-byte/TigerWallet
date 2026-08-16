package com.tigeradmin.ui.fragments

import android.content.res.Configuration
import android.graphics.Color
import android.os.Bundle
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.google.android.material.button.MaterialButton
import kotlinx.coroutines.launch

/**
 * Generic admin domain list fragment.
 *
 * Renders a RecyclerView with loading / error / empty states and light/dark theme
 * support driven by the system Configuration (AppCompatDelegate). Subclasses supply
 * the domain title, the data loader and the row renderer + optional action buttons.
 */
abstract class DomainListFragment<T> : Fragment() {

    protected abstract fun fragmentTitle(): String
    protected abstract suspend fun loadRecords(): List<T>
    protected abstract fun rowTitle(item: T): String
    protected abstract fun rowSubtitle(item: T): String
    protected open fun rowStatus(item: T): String? = null
    protected open fun primaryActionLabel(): String? = null
    protected open fun onPrimaryAction(item: T) {}
    protected open fun secondaryActionLabel(): String? = null
    protected open fun onSecondaryAction(item: T) {}

    private lateinit var container: LinearLayout
    private lateinit var progressBar: ProgressBar
    private lateinit var errorText: TextView
    private lateinit var emptyText: TextView
    private lateinit var recyclerView: RecyclerView
    private lateinit var adapter: DomainRowAdapter<T>

    private val records = mutableListOf<T>()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val ctx = requireContext()
        val isDark = (resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK) ==
            Configuration.UI_MODE_NIGHT_YES
        val bg = if (isDark) Color.parseColor("#121212") else Color.parseColor("#FAFAFA")
        val fg = if (isDark) Color.parseColor("#E6E6E6") else Color.parseColor("#1A1A1A")
        val muted = if (isDark) Color.parseColor("#9AA0A6") else Color.parseColor("#5F6368")

        val root = FrameLayout(ctx).apply {
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
            setBackgroundColor(bg)
        }

        container = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
            setPadding(24, 24, 24, 24)
        }

        val title = TextView(ctx).apply {
            text = fragmentTitle()
            textSize = 20f
            setTextColor(fg)
            setPadding(0, 0, 0, 24)
        }
        container.addView(title)

        progressBar = ProgressBar(ctx).apply {
            visibility = View.GONE
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply { gravity = Gravity.CENTER_HORIZONTAL }
        }
        container.addView(progressBar)

        errorText = TextView(ctx).apply {
            visibility = View.GONE
            setTextColor(Color.parseColor("#D93025"))
            gravity = Gravity.CENTER
            setPadding(16, 48, 16, 48)
        }
        container.addView(errorText)

        emptyText = TextView(ctx).apply {
            visibility = View.GONE
            setTextColor(muted)
            gravity = Gravity.CENTER
            text = "No records found"
            setPadding(16, 48, 16, 48)
        }
        container.addView(emptyText)

        recyclerView = RecyclerView(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
            layoutManager = LinearLayoutManager(ctx)
        }
        container.addView(recyclerView)
        root.addView(container)

        adapter = DomainRowAdapter(records, fg, muted, ::rowTitle, ::rowSubtitle, ::rowStatus,
            primaryActionLabel(), ::onPrimaryAction, secondaryActionLabel(), ::onSecondaryAction)
        recyclerView.adapter = adapter

        return root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        reload()
    }

    protected fun reload() {
        progressBar.visibility = View.VISIBLE
        errorText.visibility = View.GONE
        emptyText.visibility = View.GONE
        recyclerView.visibility = View.GONE
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val result = loadRecords()
                records.clear()
                records.addAll(result)
                adapter.notifyDataSetChanged()
                when {
                    result.isEmpty() -> emptyText.visibility = View.VISIBLE
                    else -> recyclerView.visibility = View.VISIBLE
                }
            } catch (e: Exception) {
                errorText.text = "Error: ${e.message}"
                errorText.visibility = View.VISIBLE
            } finally {
                progressBar.visibility = View.GONE
            }
        }
    }

    protected fun toast(message: String) {
        Toast.makeText(requireContext(), message, Toast.LENGTH_SHORT).show()
    }
}

/**
 * Simple RecyclerView adapter that renders one row per domain record with optional
 * primary/secondary action buttons. Theme-aware via fg/muted colors passed in.
 */
class DomainRowAdapter<T>(
    private val items: List<T>,
    private val fg: Int,
    private val muted: Int,
    private val titleOf: (T) -> String,
    private val subtitleOf: (T) -> String,
    private val statusOf: (T) -> String?,
    private val primaryLabel: String?,
    private val onPrimary: (T) -> Unit,
    private val secondaryLabel: String?,
    private val onSecondary: (T) -> Unit
) : RecyclerView.Adapter<DomainRowAdapter<T>.ViewHolder>() {

    inner class ViewHolder(view: android.view.View) : RecyclerView.ViewHolder(view) {
        val title: TextView
        val subtitle: TextView
        val status: TextView
        val primary: MaterialButton?
        val secondary: MaterialButton?

        init {
            val row = view as LinearLayout
            title = row.getChildAt(0) as TextView
            subtitle = row.getChildAt(1) as TextView
            status = row.getChildAt(2) as TextView
            val actions = row.getChildAt(3) as LinearLayout
            secondary = if (actions.childCount > 0) actions.getChildAt(0) as? MaterialButton else null
            primary = if (actions.childCount > 1) actions.getChildAt(1) as? MaterialButton else null
        }
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val ctx = parent.context
        val row = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(16, 16, 16, 16)
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
        }
        val title = TextView(ctx).apply { setTextColor(fg); textSize = 16f }
        val subtitle = TextView(ctx).apply { setTextColor(muted); textSize = 13f }
        val status = TextView(ctx).apply { setTextColor(muted); textSize = 13f }
        row.addView(title)
        row.addView(subtitle)
        row.addView(status)
        val actions = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.END
        }
        secondaryLabel?.let { label ->
            actions.addView(MaterialButton(ctx).apply { text = label })
        }
        primaryLabel?.let { label ->
            actions.addView(MaterialButton(ctx).apply { text = label })
        }
        row.addView(actions)
        return ViewHolder(row)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val item = items[position]
        holder.title.text = titleOf(item)
        holder.subtitle.text = subtitleOf(item)
        holder.status.text = statusOf(item) ?: ""
        holder.primary?.setOnClickListener { onPrimary(item) }
        holder.secondary?.setOnClickListener { onSecondary(item) }
    }

    override fun getItemCount(): Int = items.size
}
