package com.tigerwhitelabeladmin;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.LinearLayout;
import android.widget.TextView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Read-only governance contract for a single WL admin domain. Renders the
 * backend endpoints (CRUD + status/approve/reject where defined) against the
 * WL backend at http://localhost:8456. Governance records only - no fund
 * movement. Theme follows the app-wide dark-mode preference.
 */
public class DomainDetailFragment extends Fragment {
    private static final String ARG_KEY = "domain_key";
    private static final String ARG_DESC = "domain_desc";

    static final Map<String, List<String>> ENDPOINTS = new HashMap<>();
    static final Map<String, List<String>> GOV_ACTIONS = new HashMap<>();
    static {
        ENDPOINTS.put("futures", Arrays.asList("GET /futures", "POST /futures", "PUT /futures/:id", "DELETE /futures/:id"));
        GOV_ACTIONS.put("futures", Arrays.asList("PUT /futures/:id/status"));
        ENDPOINTS.put("options", Arrays.asList("GET /options", "POST /options", "PUT /options/:id", "DELETE /options/:id"));
        GOV_ACTIONS.put("options", Arrays.asList("PUT /options/:id/status"));
        ENDPOINTS.put("copy-trading", Arrays.asList("GET /copy-trading", "POST /copy-trading", "PUT /copy-trading/:id", "DELETE /copy-trading/:id"));
        GOV_ACTIONS.put("copy-trading", Arrays.asList("PUT /copy-trading/:id/status"));
        ENDPOINTS.put("convert", Arrays.asList("GET /convert", "POST /convert", "PUT /convert/:id", "DELETE /convert/:id"));
        GOV_ACTIONS.put("convert", Arrays.asList("PUT /convert/:id/status"));
        ENDPOINTS.put("onramp", Arrays.asList("GET /onramp", "POST /onramp", "PUT /onramp/:id", "DELETE /onramp/:id"));
        GOV_ACTIONS.put("onramp", Arrays.asList("POST /onramp/:id/approve", "POST /onramp/:id/reject {reason}"));
        ENDPOINTS.put("offramp", Arrays.asList("GET /offramp", "POST /offramp", "PUT /offramp/:id", "DELETE /offramp/:id"));
        GOV_ACTIONS.put("offramp", Arrays.asList("POST /offramp/:id/approve", "POST /offramp/:id/reject {reason}"));
        ENDPOINTS.put("p2p-clients", Arrays.asList("GET /p2p-clients", "POST /p2p-clients", "PUT /p2p-clients/:id", "DELETE /p2p-clients/:id"));
        GOV_ACTIONS.put("p2p-clients", Arrays.asList("PUT /p2p-clients/:id/status"));
        ENDPOINTS.put("partners", Arrays.asList("GET /partners", "POST /partners", "PUT /partners/:id", "DELETE /partners/:id"));
        GOV_ACTIONS.put("partners", Arrays.asList("PUT /partners/:id/status", "POST /partners/:id/approve", "POST /partners/:id/reject {reason}"));
        ENDPOINTS.put("rewards", Arrays.asList("GET /rewards", "POST /rewards", "PUT /rewards/:id", "DELETE /rewards/:id"));
        GOV_ACTIONS.put("rewards", Arrays.asList("PUT /rewards/:id/status"));
        ENDPOINTS.put("marketing", Arrays.asList("GET /marketing", "POST /marketing", "PUT /marketing/:id", "DELETE /marketing/:id"));
        GOV_ACTIONS.put("marketing", Arrays.asList("PUT /marketing/:id/status"));
        ENDPOINTS.put("liquidity", Arrays.asList("GET /wl-liquidity/sources", "POST /wl-liquidity/sources", "PUT /wl-liquidity/sources/:id", "DELETE /wl-liquidity/sources/:id", "GET /wl-liquidity/allocations", "POST /wl-liquidity/allocations", "GET /wl-liquidity/stats"));
        GOV_ACTIONS.put("liquidity", java.util.Collections.emptyList());
        ENDPOINTS.put("crypto-card", Arrays.asList("GET /wl-cards", "POST /wl-cards", "GET /wl-cards/transactions", "GET /wl-cards/stats"));
        GOV_ACTIONS.put("crypto-card", Arrays.asList("PUT /wl-cards/:id/status"));
        ENDPOINTS.put("bots", Arrays.asList("GET /wl-bots/operators", "POST /wl-bots/operators", "GET /wl-bots/config", "GET /wl-bots/stats"));
        GOV_ACTIONS.put("bots", Arrays.asList("PUT /wl-bots/operators/:id/status"));
        ENDPOINTS.put("kyc", Arrays.asList("GET /kyc"));
        GOV_ACTIONS.put("kyc", Arrays.asList("POST /kyc/:id/approve", "POST /kyc/:id/reject {reason}"));
        ENDPOINTS.put("tickets", Arrays.asList("GET /tickets", "GET /tickets/:id", "POST /tickets"));
        GOV_ACTIONS.put("tickets", Arrays.asList("PUT /tickets/:id/status", "POST /tickets/:id/messages", "PUT /tickets/:id/assign"));
        ENDPOINTS.put("ip-whitelist", Arrays.asList("GET /ip-whitelist", "POST /ip-whitelist"));
        GOV_ACTIONS.put("ip-whitelist", Arrays.asList("DELETE /ip-whitelist/:id"));
        ENDPOINTS.put("audit-logs", Arrays.asList("GET /audit-logs"));
        GOV_ACTIONS.put("audit-logs", Arrays.asList("POST /audit-logs/export"));
        ENDPOINTS.put("wallet-management", Arrays.asList("GET /withdrawals", "GET /fees", "POST /fees", "PUT /fees/:id", "PUT /users/:id/status"));
        GOV_ACTIONS.put("wallet-management", Arrays.asList("POST /withdrawals/:id/approve", "POST /withdrawals/:id/reject {reason}", "POST /withdrawals/:id/process"));
        ENDPOINTS.put("withdrawals", Arrays.asList("GET /withdrawals"));
        GOV_ACTIONS.put("withdrawals", Arrays.asList("POST /withdrawals/:id/approve", "POST /withdrawals/:id/reject {reason}", "POST /withdrawals/:id/process"));
        ENDPOINTS.put("rbac", Arrays.asList("GET /admin-roles", "POST /admin-roles", "GET /admin-permissions", "POST /admins/:id/role", "GET /admins/:id/permissions"));
        GOV_ACTIONS.put("rbac", Arrays.asList("DELETE /admins/:id/role/:roleId"));
    }

    static DomainDetailFragment newInstance(String key, String desc) {
        DomainDetailFragment f = new DomainDetailFragment();
        Bundle args = new Bundle();
        args.putString(ARG_KEY, key);
        args.putString(ARG_DESC, desc);
        f.setArguments(args);
        return f;
    }

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        LinearLayout root = new LinearLayout(requireContext());
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(48, 48, 48, 48);

        Bundle args = getArguments();
        String key = args != null ? args.getString(ARG_KEY, "") : "";
        String desc = args != null ? args.getString(ARG_DESC, "") : "";

        TextView title = new TextView(requireContext());
        title.setText(key);
        title.setTextSize(20f);
        title.setPadding(0, 0, 0, 32);
        root.addView(title);

        TextView subtitle = new TextView(requireContext());
        subtitle.setText(desc);
        subtitle.setTextSize(14f);
        subtitle.setPadding(0, 0, 0, 32);
        root.addView(subtitle);

        TextView crudHeader = new TextView(requireContext());
        crudHeader.setText("CRUD Endpoints");
        crudHeader.setTextSize(16f);
        root.addView(crudHeader);
        for (String ep : ENDPOINTS.getOrDefault(key, java.util.Collections.emptyList())) {
            TextView t = new TextView(requireContext());
            t.setText(ep);
            t.setTypeface(android.graphics.Typeface.MONOSPACE);
            t.setTextSize(13f);
            root.addView(t);
        }

        List<String> gov = GOV_ACTIONS.get(key);
        if (gov != null && !gov.isEmpty()) {
            TextView govHeader = new TextView(requireContext());
            govHeader.setText("Governance Actions");
            govHeader.setTextSize(16f);
            govHeader.setPadding(0, 32, 0, 16);
            root.addView(govHeader);
            for (String act : gov) {
                TextView t = new TextView(requireContext());
                t.setText(act);
                t.setTypeface(android.graphics.Typeface.MONOSPACE);
                t.setTextSize(13f);
                root.addView(t);
            }
        }

        TextView note = new TextView(requireContext());
        note.setText("No fund movement - governance records only. WL backend :8456.");
        note.setTextSize(12f);
        note.setPadding(0, 32, 0, 0);
        root.addView(note);
        return root;
    }
}
