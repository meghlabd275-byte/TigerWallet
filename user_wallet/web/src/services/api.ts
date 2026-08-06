// API Service - Connects to UserWallet Backend
import axios, { AxiosInstance } from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8105/api/v1';

class ApiService {
  private client: AxiosInstance;
  private token: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: { 'Content-Type': 'application/json' }
    });
    
    this.client.interceptors.request.use(config => {
      if (this.token) {
        config.headers.Authorization = `Bearer ${this.token}`;
      }
      return config;
    });
  }

  setToken(token: string) {
    this.token = token;
  }

  // Auth
  async login(email: string, password: string) {
    const { data } = await this.client.post('/auth/login', { email, password });
    return data;
  }

  async register(email: string, username: string, password: string) {
    const { data } = await this.client.post('/auth/register', { email, username, password });
    return data;
  }

  async getProfile() {
    const { data } = await this.client.get('/profile');
    return data;
  }

  // Wallets
  async getWallets() {
    const { data } = await this.client.get('/wallets');
    return data;
  }

  async createWallet(name: string, walletType: string, networks: string[]) {
    const { data } = await this.client.post('/wallets', { name, wallet_type: walletType, networks });
    return data;
  }

  // Transactions
  async getTransactions(params?: { network?: string; token?: string }) {
    const { data } = await this.client.get('/transactions', { params });
    return data;
  }

  async createTransaction(walletId: string, toAddress: string, amount: string, token: string, network: string) {
    const { data } = await this.client.post('/transactions', {
      wallet_id: walletId,
      to_address: toAddress,
      amount,
      token,
      network,
      type: 'send'
    });
    return data;
  }

  // Balances
  async getBalances() {
    const { data } = await this.client.get('/balances');
    return data;
  }

  async getBalance(walletId: string, token: string, network: string) {
    const { data } = await this.client.get(`/balances/${walletId}`, { params: { token, network } });
    return data;
  }

  // Prices
  async getTokenPrice(token: string, network: string) {
    const { data } = await this.client.get(`/prices/${token}`, { params: { network } });
    return data;
  }

  // Networks
  async getNetworks() {
    const { data } = await this.client.get('/networks');
    return data;
  }

  // Gas
  async getGasPrice(network: string) {
    const { data } = await this.client.get(`/network/${network}/gas`);
    return data;
  }

  // Network Status
  async getNetworkStatus(network: string) {
    const { data } = await this.client.get(`/network/${network}/status`);
    return data;
  }

  // KYC
  async getKYCStatus() {
    const { data } = await this.client.get('/kyc/status');
    return data;
  }
}

export const api = new ApiService();
export default api;
