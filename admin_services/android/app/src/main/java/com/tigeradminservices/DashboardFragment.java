package com.tigeradminservices;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;

public class DashboardFragment extends Fragment {
    private TextView tvServices, tvHealth, tvUptime, tvErrors;

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_dashboard, container, false);
        tvServices = view.findViewById(R.id.tv_services);
        tvHealth = view.findViewById(R.id.tv_health);
        tvUptime = view.findViewById(R.id.tv_uptime);
        tvErrors = view.findViewById(R.id.tv_errors);
        
        tvServices.setText("0");
        tvHealth.setText("Healthy");
        tvUptime.setText("0h");
        tvErrors.setText("0");
        
        return view;
    }
}
