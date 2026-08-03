// P2P Merchant API Service
// Production-ready API calls for merchant management

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

// API Headers with authentication
const getHeaders = () => {
  const token = localStorage.getItem('admin_token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
};

// Handle API responses
const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'An error occurred' }));
    throw new Error(error.message || `HTTP ${response.status}`);
  }
  return response.json();
};

// ==================== Types ====================

export interface MerchantTier {
  id: string;
  name: string;
  collateralAmount: number;
  maxOrderLimit: number;
  maxDailyVolume: number;
  features: string[];
  color: string;
}

export interface P2PMerchant {
  id: string;
  userId: string;
  username: string;
  email: string;
  phone: string;
  verificationLevel: 'unverified' | 'email' | 'phone' | 'kyc' | 'advanced';
  status: 'active' | 'suspended' | 'banned';
  rating: number;
  totalTrades: number;
  completionRate: number;
  totalVolume: number;
  currency: string;
  paymentMethods: string[];
  limits: {
    minOrder: number;
    maxOrder: number;
  };
  tier: string;
  collateralToken: 'USDT' | 'USDC' | 'none';
  collateralAmount: number;
  collateralTxHash?: string;
  collateralStatus: 'none' | 'pending' | 'deposited' | 'released';
  createdAt: string;
  lastActive: string;
  bannedReason?: string;
  fees: number;
}

export interface MerchantAdvertisement {
  id: string;
  merchantId: string;
  type: 'buy' | 'sell';
  crypto: string;
  fiat: string;
  price: number;
  premium: number;
  limits: {
    min: number;
    max: number;
  };
  paymentMethods: string[];
  status: 'active' | 'paused' | 'closed';
  ordersCount: number;
  completionRate: number;
  createdAt: string;
}

export interface CollateralDeposit {
  id: string;
  merchantId: string;
  merchantName: string;
  tier: string;
  amount: number;
  token: 'USDT' | 'USDC';
  txHash: string;
  status: 'pending' | 'approved' | 'rejected';
  createdAt: string;
}

export interface MerchantStats {
  totalMerchants: number;
  activeMerchants: number;
  verifiedMerchants: number;
  totalVolume: number;
  avgRating: number;
  activeAds: number;
  totalCollateral: number;
  pendingCollateral: number;
}

// ==================== API Functions ====================

// Get all merchants with filters
export const getMerchants = async (params?: {
  status?: string;
  verification?: string;
  search?: string;
  page?: number;
  limit?: number;
}): Promise<{ merchants: P2PMerchant[]; total: number }> => {
  const queryParams = new URLSearchParams();
  
  if (params?.status && params.status !== 'all') queryParams.set('status', params.status);
  if (params?.verification && params.verification !== 'all') queryParams.set('verification', params.verification);
  if (params?.search) queryParams.set('search', params.search);
  if (params?.page) queryParams.set('page', String(params.page));
  if (params?.limit) queryParams.set('limit', String(params.limit));

  const response = await fetch(`${API_BASE_URL}/p2p/merchants?${queryParams}`, {
    headers: getHeaders(),
  });
  
  return handleResponse(response);
};

// Get single merchant by ID
export const getMerchantById = async (id: string): Promise<P2PMerchant> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/${id}`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Update merchant status
export const updateMerchantStatus = async (id: string, status: string): Promise<P2PMerchant> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/${id}/status`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify({ status }),
  });
  return handleResponse(response);
};

// Update merchant tier
export const updateMerchantTier = async (id: string, tier: string): Promise<P2PMerchant> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/${id}/tier`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify({ tier }),
  });
  return handleResponse(response);
};

// Get merchant advertisements
export const getMerchantAds = async (merchantId: string): Promise<MerchantAdvertisement[]> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/${merchantId}/advertisements`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Get all advertisements
export const getAdvertisements = async (params?: {
  status?: string;
  crypto?: string;
  page?: number;
  limit?: number;
}): Promise<{ ads: MerchantAdvertisement[]; total: number }> => {
  const queryParams = new URLSearchParams();
  
  if (params?.status) queryParams.set('status', params.status);
  if (params?.crypto) queryParams.set('crypto', params.crypto);
  if (params?.page) queryParams.set('page', String(params.page));
  if (params?.limit) queryParams.set('limit', String(params.limit));

  const response = await fetch(`${API_BASE_URL}/p2p/advertisements?${queryParams}`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Update advertisement status
export const updateAdStatus = async (id: string, status: string): Promise<MerchantAdvertisement> => {
  const response = await fetch(`${API_BASE_URL}/p2p/advertisements/${id}/status`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify({ status }),
  });
  return handleResponse(response);
};

// Get tier configuration
export const getTiers = async (): Promise<MerchantTier[]> => {
  const response = await fetch(`${API_BASE_URL}/p2p/tiers`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Update tier configuration
export const updateTier = async (id: string, data: Partial<MerchantTier>): Promise<MerchantTier> => {
  const response = await fetch(`${API_BASE_URL}/p2p/tiers/${id}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse(response);
};

// Get pending collateral deposits
export const getPendingCollateral = async (): Promise<CollateralDeposit[]> => {
  const response = await fetch(`${API_BASE_URL}/p2p/collateral/pending`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Approve collateral deposit
export const approveCollateral = async (depositId: string): Promise<CollateralDeposit> => {
  const response = await fetch(`${API_BASE_URL}/p2p/collateral/${depositId}/approve`, {
    method: 'POST',
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Reject collateral deposit
export const rejectCollateral = async (depositId: string, reason: string): Promise<CollateralDeposit> => {
  const response = await fetch(`${API_BASE_URL}/p2p/collateral/${depositId}/reject`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify({ reason }),
  });
  return handleResponse(response);
};

// Release collateral
export const releaseCollateral = async (merchantId: string): Promise<P2PMerchant> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/${merchantId}/collateral/release`, {
    method: 'POST',
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Get merchant statistics
export const getMerchantStats = async (): Promise<MerchantStats> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/stats`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// Get top merchants
export const getTopMerchants = async (metric: 'volume' | 'trades', limit: number = 10): Promise<P2PMerchant[]> => {
  const response = await fetch(`${API_BASE_URL}/p2p/merchants/top?metric=${metric}&limit=${limit}`, {
    headers: getHeaders(),
  });
  return handleResponse(response);
};

// ==================== Tier Configuration ====================

// Default tier configuration
export const defaultTiers: MerchantTier[] = [
  { 
    id: 'bronze', 
    name: 'Bronze', 
    collateralAmount: 100, 
    maxOrderLimit: 1000, 
    maxDailyVolume: 5000, 
    features: ['Basic Trading', 'Email Support'], 
    color: '#cd7f32' 
  },
  { 
    id: 'silver', 
    name: 'Silver', 
    collateralAmount: 250, 
    maxOrderLimit: 5000, 
    maxDailyVolume: 25000, 
    features: ['Basic Trading', 'Priority Support', 'More Payment Methods'], 
    color: '#c0c0c0' 
  },
  { 
    id: 'gold', 
    name: 'Gold', 
    collateralAmount: 500, 
    maxOrderLimit: 15000, 
    maxDailyVolume: 75000, 
    features: ['Advanced Trading', 'Priority Support', 'All Payment Methods', 'Lower Fees'], 
    color: '#ffd700' 
  },
  { 
    id: 'platinum', 
    name: 'Platinum', 
    collateralAmount: 1000, 
    maxOrderLimit: 50000, 
    maxDailyVolume: 250000, 
    features: ['All Features', 'VIP Support', 'API Access', 'Lowest Fees'], 
    color: '#e5e4e2' 
  },
];

// Collateral requirements
export const COLLATERAL_REQUIREMENTS = {
  minCollateral: 100,
  maxCollateral: 1000,
  acceptedTokens: ['USDT', 'USDC'] as const,
};
