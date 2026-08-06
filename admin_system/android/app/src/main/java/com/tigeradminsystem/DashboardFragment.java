package com.tigeradminsystem;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;

public class DashboardFragment extends Fragment {
    private TextView tvUsers, tvConfig, tvAudit, tvSystem;

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_dashboard, container, false);
        tvUsers = view.findViewById(R.id.tv_users);
        tvConfig = view.findViewById(R.id.tv_config);
        tvAudit = view.findViewById(R.id.tv_audit);
        tvSystem = view.findViewById(R.id.tv_system);
        
        tvUsers.setText("0");
        tvConfig.setText("0");
        tvAudit.setText("0");
        tvSystem.setText("OK");
        
        return view;
    }
}
