package com.tigerwhitelabeladmin;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class DashboardFragment extends Fragment {
    private TextView tvTotalUsers, tvTotalTransactions, tvPendingKYC, tvRevenue;
    private ExecutorService executor = Executors.newSingleThreadExecutor();

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_dashboard, container, false);
        
        tvTotalUsers = view.findViewById(R.id.tv_total_users);
        tvTotalTransactions = view.findViewById(R.id.tv_total_transactions);
        tvPendingKYC = view.findViewById(R.id.tv_pending_kyc);
        tvRevenue = view.findViewById(R.id.tv_revenue);
        
        loadDashboard();
        
        return view;
    }
    
    private void loadDashboard() {
        executor.execute(() -> {
            if (getActivity() != null) {
                getActivity().runOnUiThread(() -> {
                    tvTotalUsers.setText("0");
                    tvTotalTransactions.setText("0");
                    tvPendingKYC.setText("0");
                    tvRevenue.setText("$0");
                });
            }
        });
    }
}
