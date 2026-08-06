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
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class TransactionsFragment extends Fragment {
    private ListView listView;
    private ArrayAdapter<String> adapter;
    private List<String> transactions = new ArrayList<>();
    private ExecutorService executor = Executors.newSingleThreadExecutor();

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_list, container, false);
        listView = view.findViewById(view.findViewById(R.id.list_view).getId());
        adapter = new ArrayAdapter<>(requireContext(), android.R.layout.simple_list_item_1, transactions);
        listView.setAdapter(adapter);
        loadTransactions();
        return view;
    }
    
    private void loadTransactions() {
        executor.execute(() -> {
            if (getActivity() != null) {
                getActivity().runOnUiThread(() -> {
                    transactions.clear(); transactions.add("Loading..."); adapter.notifyDataSetChanged();
                });
            }
        });
    }
}
