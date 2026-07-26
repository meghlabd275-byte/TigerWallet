import axios, { AxiosInstance, AxiosError } from 'axios';
import {
  CardData,
  Transaction,
  CreateCardRequest,
  ProcessTransactionRequest,
  UpdateLimitsRequest,
  CryptoRates,
  ApiResponse,
  CardStats,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8085/api/v1/cards';

class CardApiService {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth token to requests
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('auth_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });

    // Handle errors
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        console.error('API Error:', error.response?.data);
        return Promise.reject(error);
      }
    );
  }

  // Card Operations
  async createCard(request: CreateCardRequest): Promise<CardData> {
    const response = await this.client.post<CardData>('', request);
    return response.data;
  }

  async getCard(cardId: string): Promise<CardData> {
    const response = await this.client.get<CardData>(`/${cardId}`);
    return response.data;
  }

  async getUserCards(userId: string): Promise<CardData[]> {
    const response = await this.client.get<{ cards: CardData[] }>('', {
      params: { user_id: userId },
    });
    return response.data.cards;
  }

  async activateCard(cardId: string): Promise<CardData> {
    const response = await this.client.post<CardData>(`/${cardId}/activate`);
    return response.data;
  }

  async blockCard(cardId: string): Promise<CardData> {
    const response = await this.client.post<CardData>(`/${cardId}/block`);
    return response.data;
  }

  async freezeCard(cardId: string): Promise<CardData> {
    const response = await this.client.post<CardData>(`/${cardId}/freeze`);
    return response.data;
  }

  async unfreezeCard(cardId: string): Promise<CardData> {
    const response = await this.client.post<CardData>(`/${cardId}/unfreeze`);
    return response.data;
  }

  async cancelCard(cardId: string): Promise<CardData> {
    const response = await this.client.post<CardData>(`/${cardId}/cancel`);
    return response.data;
  }

  async updateCardLimits(cardId: string, limits: UpdateLimitsRequest['limits']): Promise<CardData> {
    const response = await this.client.put<CardData>(`/${cardId}/limits`, { limits });
    return response.data;
  }

  // Transaction Operations
  async processTransaction(request: ProcessTransactionRequest): Promise<Transaction> {
    const response = await this.client.post<Transaction>('/transactions', request);
    return response.data;
  }

  async getCardTransactions(cardId: string, days: number = 30): Promise<Transaction[]> {
    const response = await this.client.get<{ transactions: Transaction[] }>(
      `/${cardId}/transactions`,
      { params: { days } }
    );
    return response.data.transactions;
  }

  // Crypto Rates
  async getCryptoRates(): Promise<CryptoRates> {
    const response = await this.client.get<{ rates: CryptoRates }>('/rates');
    return response.data.rates;
  }

  // Helper to calculate stats
  calculateStats(cards: CardData[]): CardStats {
    const stats: CardStats = {
      totalCards: cards.length,
      activeCards: 0,
      blockedCards: 0,
      totalSpent: 0,
      monthlySpent: 0,
    };

    cards.forEach((card) => {
      if (card.status === 'ACTIVE') {
        stats.activeCards++;
      } else if (card.status === 'BLOCKED') {
        stats.blockedCards++;
      }
      stats.totalSpent += card.daily_spent;
      stats.monthlySpent += card.monthly_spent;
    });

    return stats;
  }

  // Format currency
  formatCurrency(amount: number, currency: string = 'USD'): string {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency,
    }).format(amount / 100);
  }

  // Format date
  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  // Format time
  formatTime(dateString: string): string {
    return new Date(dateString).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
    });
  }
}

export const cardApi = new CardApiService();
export default cardApi;
