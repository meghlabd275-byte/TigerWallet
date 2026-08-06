package com.tigermasteradmin;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;
import org.json.JSONObject;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class DashboardFragment extends Fragment {
    private TextView tvTotalWhiteLabels, tvTotalUsers, tvTotalTransactions, tvPendingApprovals;
    private ExecutorService executor = Executors.newSingleThreadExecutor();

    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_dashboard, container, false);
        
        tvTotalWhiteLabels = view.findViewById(R.id.tv_total_white_labels);
        tvTotalUsers = view.findViewById(R.id.tv_total_users);
        tvTotalTransactions = view.findViewById(R.id.tv_total_transactions);
        tvPendingApprovals = view.findViewById(R.id.tv_pending_approvals);
        
        loadDashboard();
        
        return view;
    }
    
    private void loadDashboard() {
        executor.execute(() -> {
            try {
                String url = TigerMasterAdminApp.getInstance().getBaseURL() + "/api/v1/dashboard/stats";
                // Simulated - in production use proper HTTP client
                if (getActivity() != null) {
                    getActivity().runOnUiThread(() -> {
                        tvTotalWhiteLabels.setText("0");
                        tvTotalUsers.setText("0");
                        tvTotalTransactions.setText("0");
                        tvPendingApprovals.setText("0");
                    });
                }
            } catch (Exception e) {
                e.printStackTrace();
            }
        });
    }
}
