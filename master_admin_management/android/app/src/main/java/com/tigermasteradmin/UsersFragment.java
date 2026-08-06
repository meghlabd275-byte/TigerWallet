package com.tigermasteradmin;

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

public class UsersFragment extends Fragment {
    private ListView listView;
    private ArrayAdapter<String> adapter;
    private List<String> users = new ArrayList<>();
    private ExecutorService executor = Executors.newSingleThreadExecutor();

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_list, container, false);
        
        listView = view.findViewById(view.findViewById(R.id.list_view).getId());
        adapter = new ArrayAdapter<>(requireContext(), android.R.layout.simple_list_item_1, users);
        listView.setAdapter(adapter);
        
        loadUsers();
        
        return view;
    }
    
    private void loadUsers() {
        executor.execute(() -> {
            if (getActivity() != null) {
                getActivity().runOnUiThread(() -> {
                    users.clear();
                    users.add("Loading...");
                    adapter.notifyDataSetChanged();
                });
            }
        });
    }
}
