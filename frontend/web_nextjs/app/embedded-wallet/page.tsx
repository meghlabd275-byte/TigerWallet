'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider'

interface SDKConfig {
  theme: 'light' | 'dark';
  accentColor: string;
  language: string;
}

const LANGUAGES = ['en', 'es', 'fr', 'de', 'zh', 'ja', 'ko', 'pt', 'ru', 'ar'];

export default function EmbeddedWalletPage() {
  const { isDark } = useTheme()
  const [config, setConfig] = useState<SDKConfig>({
    theme: 'dark',
    accentColor: '#3B82F6',
    language: 'en',
  });
  const [showCode, setShowCode] = useState(false);

  const codeSnippet = `
// Install
npm install @tigerwallet/embedded-sdk

// React Component
import { TigerWalletEmbed } from '@tigerwallet/embedded-sdk';

function MyApp() {
  return (
    <TigerWalletEmbed
      theme="${config.theme}"
      accentColor="${config.accentColor}"
      language="${config.language}"
      onConnect={(address) => console.log(address)}
    />
  );
}
`;

  return (
    <div className={`'min-h-screen bg-gradient-to-br' ${isDark ? 'from-slate-900' : 'from-slate-50'} ${isDark ? 'to-slate-800' : 'to-slate-100'} 'p-8'`}>
      <div className="max-w-6xl mx-auto">
        <h1 className={`'text-4xl font-bold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-2'`}>Embedded Wallet SDK</h1>
        <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} 'mb-8'`}>Integrate TigerWallet into your app</p>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Configuration */}
          <div>
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-2xl p-6 border' ${isDark ? 'border-slate-700' : 'border-slate-200'} 'mb-6'`}>
              <h2 className={`'text-xl font-semibold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-4'`}>Customize Widget</h2>
              
              <div className="space-y-4">
                <div>
                  <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>Theme</label>
                  <select
                    value={config.theme}
                    onChange={(e) => setConfig(prev => ({ ...prev, theme: e.target.value as 'light' | 'dark' }))}
                    className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-slate-900'}`}
                  >
                    <option value="dark">Dark</option>
                    <option value="light">Light</option>
                  </select>
                </div>

                <div>
                  <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>Accent Color</label>
                  <div className="flex gap-2">
                    {['#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899'].map(color => (
                      <button
                        key={color}
                        onClick={() => setConfig(prev => ({ ...prev, accentColor: color }))}
                        className={`w-10 h-10 rounded-lg ${config.accentColor === color ? 'ring-2 ring-white' : ''}`}
                        style={{ backgroundColor: color }}
                      />
                    ))}
                  </div>
                </div>

                <div>
                  <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>Language</label>
                  <select
                    value={config.language}
                    onChange={(e) => setConfig(prev => ({ ...prev, language: e.target.value }))}
                    className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-slate-900'}`}
                  >
                    {LANGUAGES.map(lang => (
                      <option key={lang} value={lang}>{lang.toUpperCase()}</option>
                    ))}
                  </select>
                </div>
              </div>
            </div>

            <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-2xl p-6 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
              <h2 className={`'text-xl font-semibold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-4'`}>Preview</h2>
              <div 
                className={`p-4 rounded-xl border ${config.theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white border-slate-200'}`}
                style={{ accentColor: config.accentColor }}
              >
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 rounded-full" style={{ backgroundColor: config.accentColor }}></div>
                  <div>
                    <p className={`font-medium ${config.theme === 'dark' ? 'text-white' : 'text-gray-900'}`}>TigerWallet</p>
                    <p className={`text-sm ${config.theme === 'dark' ? 'text-slate-400' : 'text-gray-500'}`}>Connect your wallet</p>
                  </div>
                </div>
                <button 
                  className={`'w-full py-2 rounded-lg' ${isDark ? 'text-white' : 'text-slate-900'} 'font-medium'`}
                  style={{ backgroundColor: config.accentColor }}
                >
                  Connect Wallet
                </button>
              </div>
            </div>
          </div>

          {/* Code */}
          <div>
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-2xl p-6 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
              <div className="flex items-center justify-between mb-4">
                <h2 className={`'text-xl font-semibold' ${isDark ? 'text-white' : 'text-slate-900'}`}>Integration Code</h2>
                <button
                  onClick={() => setShowCode(!showCode)}
                  className="text-blue-400 hover:text-blue-300"
                >
                  {showCode ? 'Hide' : 'Show'} Code
                </button>
              </div>
              
              {showCode && (
                <pre className={`${isDark ? 'bg-slate-900' : 'bg-slate-50'} 'p-4 rounded-lg overflow-x-auto text-sm' ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>
                  <code>{codeSnippet}</code>
                </pre>
              )}
            </div>

            <div className="mt-6 space-y-4">
              <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
                <h3 className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium mb-2'`}>Supported Platforms</h3>
                <div className="flex gap-2 flex-wrap">
                  {['React', 'Vue', 'Angular', 'Next.js', 'iOS', 'Android', 'Unity'].map(platform => (
                    <span key={platform} className={`'px-3 py-1' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'rounded-full text-sm' ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>
                      {platform}
                    </span>
                  ))}
                </div>
              </div>

              <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
                <h3 className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium mb-2'`}>Features</h3>
                <ul className={`'space-y-2' ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                  <li>✓ Wallet connection</li>
                  <li>✓ Transaction signing</li>
                  <li>✓ Multi-chain support</li>
                  <li>✓ Gasless transactions</li>
                  <li>✓ Custom theming</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
