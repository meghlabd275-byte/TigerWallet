package com.tigeradminconsole;

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

public class MetricsFragment extends Fragment {
    private ListView listView;
    private ArrayAdapter<String> adapter;
    private List<String> metrics = new ArrayList<>();

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_list, container, false);
        listView = view.findViewById(R.id.list_view);
        adapter = new ArrayAdapter<>(requireContext(), android.R.layout.simple_list_item_1, metrics);
        listView.setAdapter(adapter);
        metrics.clear(); metrics.add("Loading..."); adapter.notifyDataSetChanged();
        return view;
    }
}
