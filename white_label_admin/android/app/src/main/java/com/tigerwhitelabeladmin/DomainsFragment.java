package com.tigerwhitelabeladmin;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.ListView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;
import java.util.Arrays;
import java.util.List;

/**
 * Index of the 11 WL admin domain screens. Selecting one opens
 * {@link DomainDetailFragment}, which renders the read-only governance
 * contract for that domain (no fund movement). Mirrors the WL web/ios/desktop
 * parity list: futures, options, copy-trading, convert, onramp, offramp,
 * p2p-clients, partners, rewards, marketing, RBAC.
 */
public class DomainsFragment extends Fragment {
    static final List<String[]> DOMAINS = Arrays.asList(
        new String[]{"futures", "Futures positions governance (CRUD + status)"},
        new String[]{"options", "Options contracts governance (CRUD + status)"},
        new String[]{"copy-trading", "Copy-trading configs governance (CRUD + status)"},
        new String[]{"convert", "Convert orders governance (CRUD + status)"},
        new String[]{"onramp", "On-ramp orders governance (approve/reject)"},
        new String[]{"offramp", "Off-ramp orders governance (approve/reject)"},
        new String[]{"p2p-clients", "P2P clients governance (CRUD + status)"},
        new String[]{"partners", "Partners governance (status + approve/reject)"},
        new String[]{"rewards", "Reward campaigns governance (CRUD + status)"},
        new String[]{"marketing", "Marketing campaigns governance (CRUD + status)"},
        new String[]{"rbac", "Admin roles & permissions governance"}
    );

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_list, container, false);
        ListView listView = view.findViewById(R.id.list_view);
        String[] labels = new String[DOMAINS.size()];
        for (int i = 0; i < DOMAINS.size(); i++) labels[i] = DOMAINS.get(i)[0];
        ArrayAdapter<String> adapter = new ArrayAdapter<>(requireContext(), android.R.layout.simple_list_item_1, labels);
        listView.setAdapter(adapter);
        listView.setOnItemClickListener((parent, v, position, id) -> {
            String[] entry = DOMAINS.get(position);
            DomainDetailFragment detail = DomainDetailFragment.newInstance(entry[0], entry[1]);
            requireActivity().getSupportFragmentManager().beginTransaction()
                .replace(R.id.fragment_container, detail)
                .addToBackStack(null)
                .commit();
        });
        return view;
    }
}
