package com.tigeradminconsole;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;

public class DashboardFragment extends Fragment {
    private TextView tvServices, tvHealth, tvRequests, tvErrors;

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_dashboard, container, false);
        tvServices = view.findViewById(R.id.tv_services);
        tvHealth = view.findViewById(R.id.tv_health);
        tvRequests = view.findViewById(R.id.tv_requests);
        tvErrors = view.findViewById(R.id.tv_errors);
        
        tvServices.setText("0");
        tvHealth.setText("Healthy");
        tvRequests.setText("0");
        tvErrors.setText("0");
        
        return view;
    }
}
