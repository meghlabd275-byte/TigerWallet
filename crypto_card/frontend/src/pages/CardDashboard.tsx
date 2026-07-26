import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { cardApi } from '../services/cardApi';
import { CardData, CardStats } from '../types';
import CardComponent from '../components/CardComponent';
import { 
  CreditCard, 
  Plus, 
  RefreshCw, 
  Moon, 
  Sun, 
  Wallet,
  TrendingUp,
  AlertTriangle,
  Search,
  Filter,
  Settings,
  Bell
} from 'lucide-react';

export const CardDashboard: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const isDark = theme === 'dark';

  const [cards, setCards] = useState<CardData[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<CardStats | null>(null);
  const [selectedCard, setSelectedCard] = useState<CardData | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  // Colors based on theme
  const colors = isDark
    ? {
        bgPrimary: 'bg-[#0a0a0f]',
        bgSecondary: 'bg-[#12121a]',
        bgTertiary: 'bg-[#1a1a24]',
        bgCard: 'bg-[#16161f]',
        bgHover: 'bg-[#1e1e2a]',
        textPrimary: 'text-white',
        textSecondary: 'text-[#a0a0b0]',
        textMuted: 'text-[#6b6b7b]',
        border: 'border-[#2a2a3a]',
        borderLight: 'border-[#3a3a4a]',
        accent: 'text-amber-500',
        accentBg: 'bg-amber-500',
      }
    : {
        bgPrimary: 'bg-white',
        bgSecondary: 'bg-slate-50',
        bgTertiary: 'bg-slate-100',
        bgCard: 'bg-white',
        bgHover: 'bg-slate-100',
        textPrimary: 'text-slate-900',
        textSecondary: 'text-slate-600',
        textMuted: 'text-slate-400',
        border: 'border-slate-200',
        borderLight: 'border-slate-300',
        accent: 'text-amber-600',
        accentBg: 'bg-amber-600',
      };

  useEffect(() => {
    loadCards();
  }, []);

  const loadCards = async () => {
    try {
      setLoading(true);
      // In production, get user ID from auth
      const userId = 'user_12345';
      const userCards = await cardApi.getUserCards(userId);
      setCards(userCards);
      setStats(cardApi.calculateStats(userCards));
    } catch (error) {
      console.error('Failed to load cards:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleActivateCard = async (cardId: string) => {
    try {
      await cardApi.activateCard(cardId);
      await loadCards();
    } catch (error) {
      console.error('Failed to activate card:', error);
    }
  };

  const handleBlockCard = async (cardId: string) => {
    try {
      await cardApi.blockCard(cardId);
      await loadCards();
    } catch (error) {
      console.error('Failed to block card:', error);
    }
  };

  const handleFreezeCard = async (cardId: string) => {
    try {
      await cardApi.freezeCard(cardId);
      await loadCards();
    } catch (error) {
      console.error('Failed to freeze card:', error);
    }
  };

  const handleUnfreezeCard = async (cardId: string) => {
    try {
      await cardApi.unfreezeCard(cardId);
      await loadCards();
    } catch (error) {
      console.error('Failed to unfreeze card:', error);
    }
  };

  const filteredCards = cards.filter((card) => {
    const matchesSearch = card.masked_number.toLowerCase().includes(searchTerm.toLowerCase()) ||
      card.card_holder_name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesFilter = filterStatus === 'all' || card.status === filterStatus;
    return matchesSearch && matchesFilter;
  });

  return (
    <div className={`min-h-screen ${colors.bgPrimary} ${colors.textPrimary}`}>
      {/* Header */}
      <header className={`${colors.bgSecondary} border-b ${colors.border} sticky top-0 z-50`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo */}
            <div className="flex items-center gap-3">
              <div className={`w-10 h-10 ${colors.accentBg} rounded-xl flex items-center justify-center`}>
                <Wallet className="w-6 h-6 text-white" />
              </div>
              <span className={`text-xl font-bold ${colors.textPrimary}`}>TigerCard</span>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-4">
              <button className={`p-2 rounded-lg ${colors.bgHover} ${colors.textSecondary} hover:${colors.accent} transition-colors`}>
                <Bell className="w-5 h-5" />
              </button>
              <button className={`p-2 rounded-lg ${colors.bgHover} ${colors.textSecondary} hover:${colors.accent} transition-colors`}>
                <Settings className="w-5 h-5" />
              </button>
              <button
                onClick={toggleTheme}
                className={`p-2 rounded-lg ${colors.bgHover} ${colors.textSecondary} hover:${colors.accent} transition-colors`}
              >
                {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats Cards */}
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
            <div className={`${colors.bgCard} border ${colors.border} rounded-xl p-4`}>
              <div className="flex items-center gap-3">
                <div className={`p-2 ${colors.accentBg} rounded-lg`}>
                  <CreditCard className="w-5 h-5 text-white" />
                </div>
                <div>
                  <p className={`text-sm ${colors.textMuted}`}>Total Cards</p>
                  <p className={`text-2xl font-bold ${colors.textPrimary}`}>{stats.totalCards}</p>
                </div>
              </div>
            </div>
            <div className={`${colors.bgCard} border ${colors.border} rounded-xl p-4`}>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-500 rounded-lg">
                  <TrendingUp className="w-5 h-5 text-white" />
                </div>
                <div>
                  <p className={`text-sm ${colors.textMuted}`}>Active Cards</p>
                  <p className={`text-2xl font-bold ${colors.textPrimary}`}>{stats.activeCards}</p>
                </div>
              </div>
            </div>
            <div className={`${colors.bgCard} border ${colors.border} rounded-xl p-4`}>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-red-500 rounded-lg">
                  <AlertTriangle className="w-5 h-5 text-white" />
                </div>
                <div>
                  <p className={`text-sm ${colors.textMuted}`}>Blocked</p>
                  <p className={`text-2xl font-bold ${colors.textPrimary}`}>{stats.blockedCards}</p>
                </div>
              </div>
            </div>
            <div className={`${colors.bgCard} border ${colors.border} rounded-xl p-4`}>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-500 rounded-lg">
                  <Wallet className="w-5 h-5 text-white" />
                </div>
                <div>
                  <p className={`text-sm ${colors.textMuted}`}>Monthly Spent</p>
                  <p className={`text-2xl font-bold ${colors.textPrimary}`}>
                    {cardApi.formatCurrency(stats.monthlySpent)}
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Filters */}
        <div className={`flex flex-col sm:flex-row justify-between items-center gap-4 mb-6`}>
          <div className="flex items-center gap-4 w-full sm:w-auto">
            <div className={`relative flex-1 sm:w-64`}>
              <Search className={`absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 ${colors.textMuted}`} />
              <input
                type="text"
                placeholder="Search cards..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className={`w-full pl-10 pr-4 py-2 ${colors.bgTertiary} border ${colors.border} rounded-lg ${colors.textPrimary} placeholder:${colors.textMuted} focus:outline-none focus:ring-2 focus:ring-amber-500`}
              />
            </div>
            <div className="relative">
              <Filter className={`absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 ${colors.textMuted}`} />
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                className={`pl-10 pr-8 py-2 ${colors.bgTertiary} border ${colors.border} rounded-lg ${colors.textPrimary} focus:outline-none focus:ring-2 focus:ring-amber-500 appearance-none`}
              >
                <option value="all">All Status</option>
                <option value="ACTIVE">Active</option>
                <option value="PENDING">Pending</option>
                <option value="BLOCKED">Blocked</option>
                <option value="FROZEN">Frozen</option>
              </select>
            </div>
          </div>
          <button
            className={`flex items-center gap-2 px-4 py-2 ${colors.accentBg} text-white rounded-lg hover:opacity-90 transition-opacity`}
          >
            <Plus className="w-4 h-4" />
            <span>New Card</span>
          </button>
        </div>

        {/* Cards Grid */}
        {loading ? (
          <div className="flex justify-center items-center py-20">
            <RefreshCw className={`w-8 h-8 animate-spin ${colors.accent}`} />
          </div>
        ) : filteredCards.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredCards.map((card) => (
              <CardComponent
                key={card.card_id}
                card={card}
                onActivate={handleActivateCard}
                onBlock={handleBlockCard}
                onFreeze={handleFreezeCard}
                onUnfreeze={handleUnfreezeCard}
                onSelect={setSelectedCard}
              />
            ))}
          </div>
        ) : (
          <div className={`text-center py-20 ${colors.textSecondary}`}>
            <CreditCard className="w-16 h-16 mx-auto mb-4 opacity-50" />
            <p className="text-lg">No cards found</p>
            <p className={`text-sm ${colors.textMuted}`}>Create your first crypto card to get started</p>
          </div>
        )}
      </main>
    </div>
  );
};

export default CardDashboard;
