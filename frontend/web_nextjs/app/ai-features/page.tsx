'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface AIFeature {
  id: string;
  name: string;
  description: string;
  icon: string;
  enabled: boolean;
}

const AI_FEATURES: AIFeature[] = [
  { id: 'signals', name: 'AI Trading Signals', description: 'Get AI-powered trading recommendations', icon: '📈', enabled: true },
  { id: 'rebalance', name: 'Portfolio Rebalancing', description: 'AI automatically rebalances your portfolio', icon: '⚖️', enabled: false },
  { id: 'whale', name: 'Whale Tracking', description: 'Follow smart money movements', icon: '🐋', enabled: false },
  { id: 'gas', name: 'Gas Prediction', description: 'Predict optimal gas times', icon: '⛽', enabled: true },
  { id: 'airdrop', name: 'Airdrop Hunter', description: 'Detect potential airdrop opportunities', icon: '🎯', enabled: false },
  { id: 'tax', name: 'Tax Loss Harvesting', description: 'Optimize tax with AI', icon: '📊', enabled: false },
];

export default function AIFeaturesPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [features, setFeatures] = useState(AI_FEATURES);
  const [query, setQuery] = useState('');
  const [response, setResponse] = useState<string | null>(null);

  const toggleFeature = (id: string) => {
    setFeatures(prev => prev.map(f => 
      f.id === id ? { ...f, enabled: !f.enabled } : f
    ));
  };

  const askAI = () => {
    if (!query) return;
    setResponse("Based on current market analysis, ETH shows strong support at $3,200. Consider DCAing into positions. Gas fees are currently low at 15 gwei.");
  };

  return (
    <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
      <div className="max-w-4xl mx-auto">
        <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>AI Features</h1>
        <p className={`mb-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>AI-powered insights for better decisions</p>

        {/* AI Chat */}
        <div className={`rounded-2xl p-6 border mb-8 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>🤖 AI Assistant</h2>
          <div className="flex gap-4">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Ask about market, gas, tokens..."
              className={`flex-1 border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`}
            />
            <button onClick={askAI} className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg font-medium">
              Ask
            </button>
          </div>
          {response && (
            <div className={`mt-4 p-4 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
              <p className={isDark ? 'text-white' : 'text-slate-900'}>{response}</p>
            </div>
          )}
        </div>

        {/* Feature Toggles */}
        <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>AI Features</h2>
        <div className="grid gap-4">
          {features.map(feature => (
            <div key={feature.id} className={`rounded-xl p-4 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-3xl">{feature.icon}</span>
                  <div>
                    <h3 className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{feature.name}</h3>
                    <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{feature.description}</p>
                  </div>
                </div>
                <button
                  onClick={() => toggleFeature(feature.id)}
                  className={`w-14 h-8 rounded-full transition-colors ${feature.enabled ? 'bg-green-500' : 'bg-slate-600'}`}
                >
                  <div className={`w-6 h-6 bg-white rounded-full shadow transition-transform ${feature.enabled ? 'translate-x-7' : 'translate-x-1'}`} />
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
