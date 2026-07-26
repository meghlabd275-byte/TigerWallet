import React from 'react';
import { CardData, CardStatus, CardNetwork } from '../types';
import { useTheme } from '../../hooks/useTheme';
import { CreditCard, Wifi, Globe, Lock, AlertCircle, CheckCircle, XCircle, Pause } from 'lucide-react';

interface CardComponentProps {
  card: CardData;
  onActivate?: (cardId: string) => void;
  onBlock?: (cardId: string) => void;
  onFreeze?: (cardId: string) => void;
  onUnfreeze?: (cardId: string) => void;
  onCancel?: (cardId: string) => void;
  onSelect?: (cardId: string) => void;
}

export const CardComponent: React.FC<CardComponentProps> = ({
  card,
  onActivate,
  onBlock,
  onFreeze,
  onUnfreeze,
  onCancel,
  onSelect,
}) => {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const getStatusColor = (status: CardStatus) => {
    switch (status) {
      case 'ACTIVE':
        return 'text-green-500';
      case 'BLOCKED':
        return 'text-red-500';
      case 'FROZEN':
        return 'text-yellow-500';
      case 'PENDING':
        return 'text-blue-500';
      case 'EXPIRED':
        return 'text-gray-500';
      case 'CANCELLED':
        return 'text-red-400';
      default:
        return 'text-gray-500';
    }
  };

  const getStatusIcon = (status: CardStatus) => {
    switch (status) {
      case 'ACTIVE':
        return <CheckCircle className="w-4 h-4" />;
      case 'BLOCKED':
        return <XCircle className="w-4 h-4" />;
      case 'FROZEN':
        return <Pause className="w-4 h-4" />;
      case 'PENDING':
        return <AlertCircle className="w-4 h-4" />;
      default:
        return <AlertCircle className="w-4 h-4" />;
    }
  };

  const getNetworkGradient = (network: CardNetwork) => {
    switch (network) {
      case 'VISA':
        return 'from-blue-600 to-blue-800';
      case 'MASTERCARD':
        return 'from-red-600 to-red-800';
      case 'AMEX':
        return 'from-green-600 to-green-800';
      case 'UNIONPAY':
        return 'from-blue-500 to-cyan-600';
      default:
        return 'from-amber-500 to-orange-600';
    }
  };

  return (
    <div
      onClick={() => onSelect?.(card.card_id)}
      className={`
        relative overflow-hidden rounded-2xl p-6 cursor-pointer
        transition-all duration-300 hover:scale-[1.02] hover:shadow-xl
        bg-gradient-to-br ${getNetworkGradient(card.network)}
        ${isDark ? 'text-white' : 'text-white'}
      `}
    >
      {/* Background Pattern */}
      <div className="absolute inset-0 opacity-10">
        <div className="absolute top-0 right-0 w-64 h-64 rounded-full transform translate-x-20 -translate-y-20 bg-white"></div>
        <div className="absolute bottom-0 left-0 w-48 h-48 rounded-full transform -translate-x-16 translate-y-16 bg-white"></div>
      </div>

      {/* Card Content */}
      <div className="relative z-10">
        {/* Header */}
        <div className="flex justify-between items-start mb-8">
          <div>
            <h3 className="text-lg font-bold tracking-wide">{card.card_holder_name}</h3>
            <p className="text-xs opacity-80 mt-1">{card.card_type}</p>
          </div>
          <div className={`flex items-center gap-1 ${getStatusColor(card.status)}`}>
            {getStatusIcon(card.status)}
            <span className="text-xs font-medium">{card.status}</span>
          </div>
        </div>

        {/* Card Number */}
        <div className="mb-6">
          <p className="text-xl font-mono tracking-widest letter-spacing-2">
            {card.masked_number}
          </p>
        </div>

        {/* Footer */}
        <div className="flex justify-between items-end">
          <div>
            <p className="text-xs opacity-70 mb-1">Expires</p>
            <p className="font-mono">
              {String(card.expiry_month).padStart(2, '0')}/{card.expiry_year}
            </p>
          </div>
          <div className="flex gap-3">
            {card.contactless_enabled && (
              <Wifi className="w-5 h-5 opacity-80" />
            )}
            {card.online_payments_enabled && (
              <Globe className="w-5 h-5 opacity-80" />
            )}
            <Lock className="w-5 h-5 opacity-80" />
          </div>
        </div>

        {/* Network Logo */}
        <div className="absolute top-6 right-6">
          <CreditCard className="w-8 h-8 opacity-50" />
        </div>
      </div>

      {/* Action Buttons */}
      <div className="relative z-20 mt-4 flex gap-2">
        {card.status === 'PENDING' && onActivate && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onActivate(card.card_id);
            }}
            className="px-3 py-1 text-xs bg-green-500 hover:bg-green-600 rounded-full transition-colors"
          >
            Activate
          </button>
        )}
        {card.status === 'ACTIVE' && onFreeze && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onFreeze(card.card_id);
            }}
            className="px-3 py-1 text-xs bg-yellow-500 hover:bg-yellow-600 rounded-full transition-colors"
          >
            Freeze
          </button>
        )}
        {card.status === 'FROZEN' && onUnfreeze && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onUnfreeze(card.card_id);
            }}
            className="px-3 py-1 text-xs bg-blue-500 hover:bg-blue-600 rounded-full transition-colors"
          >
            Unfreeze
          </button>
        )}
        {card.status === 'ACTIVE' && onBlock && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onBlock(card.card_id);
            }}
            className="px-3 py-1 text-xs bg-red-500 hover:bg-red-600 rounded-full transition-colors"
          >
            Block
          </button>
        )}
      </div>
    </div>
  );
};

export default CardComponent;
