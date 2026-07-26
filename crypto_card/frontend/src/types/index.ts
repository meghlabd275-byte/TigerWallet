// Card Types
export type CardType = 'VIRTUAL' | 'PHYSICAL' | 'VIRTUAL_ONE_TIME' | 'METAL';
export type CardStatus = 'PENDING' | 'ACTIVE' | 'BLOCKED' | 'EXPIRED' | 'CANCELLED' | 'FROZEN';
export type CardNetwork = 'VISA' | 'MASTERCARD' | 'AMEX' | 'UNIONPAY';
export type TransactionType = 'PURCHASE' | 'WITHDRAWAL' | 'REFUND' | 'TRANSFER' | 'TOP_UP' | 'FEE';
export type TransactionStatus = 'PENDING' | 'COMPLETED' | 'FAILED' | 'CANCELLED' | 'FLAGGED';

export interface CardHolder {
  user_id: string;
  name: string;
  email: string;
  phone: string;
  billing_address: string;
  country: string;
  city: string;
  postal_code: string;
  kyc_level: string;
  risk_level: number;
}

export interface CardData {
  card_id: string;
  user_id: string;
  masked_number: string;
  last_four: string;
  expiry_month: number;
  expiry_year: number;
  card_type: CardType;
  status: CardStatus;
  network: CardNetwork;
  currency: string;
  card_holder_name: string;
  billing_address: string;
  daily_limit: number;
  monthly_limit: number;
  daily_spent: number;
  monthly_spent: number;
  max_single_transaction: number;
  min_single_transaction: number;
  contactless_enabled: boolean;
  online_payments_enabled: boolean;
  international_enabled: boolean;
  created_at: string;
  updated_at: string;
  expires_at: string;
}

export interface Transaction {
  transaction_id: string;
  card_id: string;
  user_id: string;
  type: TransactionType;
  status: TransactionStatus;
  currency: string;
  amount: number;
  fee: number;
  crypto_amount?: number;
  crypto_currency?: string;
  merchant_id: string;
  merchant_name: string;
  merchant_category: string;
  terminal_id: string;
  location: string;
  ip_address: string;
  description: string;
  reference_id: string;
  authorization_code: string;
  timestamp: string;
  settled_at?: string;
  blockchain_tx_hash?: string;
  risk_score: number;
  risk_reason: string;
}

export interface CardLimits {
  daily_limit: number;
  monthly_limit: number;
  max_single_transaction: number;
  min_single_transaction: number;
  daily_withdrawal_limit: number;
  monthly_withdrawal_limit: number;
}

export interface CreateCardRequest {
  user_id: string;
  card_type: CardType;
  network: CardNetwork;
  currency: string;
  holder: CardHolder;
}

export interface ProcessTransactionRequest {
  card_id: string;
  user_id: string;
  type: TransactionType;
  amount: number;
  currency: string;
  crypto_currency?: string;
  merchant_id: string;
  merchant_name: string;
  merchant_category: string;
  location: string;
  ip_address: string;
  description: string;
}

export interface UpdateLimitsRequest {
  card_id: string;
  limits: CardLimits;
}

export interface CryptoRates {
  [key: string]: number;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface CardStats {
  totalCards: number;
  activeCards: number;
  blockedCards: number;
  totalSpent: number;
  monthlySpent: number;
}
