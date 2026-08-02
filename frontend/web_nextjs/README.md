# TigerWallet Web App

## Overview

TigerWallet Web App is a fully-featured Progressive Web Application (PWA) built with Next.js 14, React, and TypeScript.

## Features

### Core Features
- Multi-chain wallet management (100+ networks)
- Token swap across DEXs
- Cross-chain bridging
- Staking
- NFT management
- Trading bots dashboard
- Token listing application
- Fiat on/off ramp
- Real-time analytics

### Security Features
- Biometric authentication
- Passkey support
- Hardware wallet integration
- Multi-sig wallet support
- Social recovery
- Account abstraction

## Technology Stack

- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS + Material UI
- **State Management**: React Context + Zustand
- **Blockchain**: ethers.js, web3.js
- **API**: REST + GraphQL

## Project Structure

```
web_nextjs/
├── app/                    # Next.js App Router pages
│   ├── page.tsx           # Home page
│   ├── wallet/            # Wallet pages
│   ├── swap/              # Swap pages
│   ├── bridge/            # Bridge pages
│   ├── staking/            # Staking pages
│   ├── nft-marketplace/   # NFT pages
│   ├── listing/           # Token listing
│   ├── bot_dashboard/     # Bot management
│   ├── fiat-ramp/         # Fiat transactions
│   ├── admin_listing/     # Admin panel
│   ├── super_admin/       # Super admin
│   └── components/        # Shared components
├── public/                # Static assets
│   ├── manifest.json     # PWA manifest
│   ├── icons/            # PWA icons
│   └── images/           # Images
└── src/                  # Source files
    ├── components/        # React components
    ├── hooks/            # Custom hooks
    ├── lib/             # Utility functions
    ├── services/        # API services
    └── types/            # TypeScript types
```

## Installation

```bash
# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Start production server
npm start
```

## Environment Variables

```env
# API URLs
NEXT_PUBLIC_API_URL=http://localhost:8097
NEXT_PUBLIC_WS_URL=ws://localhost:8095

# Blockchain RPC URLs
NEXT_PUBLIC_ETH_RPC=https://eth.llamarpc.com
NEXT_PUBLIC_BSC_RPC=https://bsc-dataseed.binance.org
NEXT_PUBLIC_POLYGON_RPC=https://polygon-rpc.com

# WalletConnect
NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID=your_project_id

# Analytics
NEXT_PUBLIC_GA_TRACKING_ID=G-XXXXXXXXXX
```

## PWA Configuration

### manifest.json

```json
{
  "name": "TigerWallet",
  "short_name": "TigerWallet",
  "description": "Enterprise Web3 Wallet - Trade, Swap, Stake across 100+ chains",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#000000",
  "theme_color": "#f97316",
  "orientation": "portrait-primary",
  "icons": [
    {
      "src": "/icons/icon-192x192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icons/icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

### Service Worker

```javascript
// public/sw.js
const CACHE_NAME = 'tigerwallet-v1';
const urlsToCache = [
  '/',
  '/index.html',
  '/static/js/main.js',
  '/static/css/main.css'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(urlsToCache))
  );
});

self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request)
      .then((response) => {
        if (response) {
          return response;
        }
        return fetch(event.request);
      })
  );
});
```

## Deployment

### Vercel

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel --prod
```

### Docker

```dockerfile
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --only=production

COPY . .

RUN npm run build

EXPOSE 3000

CMD ["npm", "start"]
```

### Nginx

```nginx
server {
    listen 80;
    server_name tigerwallet.io;
    
    root /var/www/tigerwallet;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    location /api/ {
        proxy_pass http://localhost:8097;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
    
    location /ws/ {
        proxy_pass ws://localhost:8095;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Browser Support

| Browser | Version |
|---------|----------|
| Chrome | 90+ |
| Firefox | 88+ |
| Safari | 14+ |
| Edge | 90+ |
| Opera | 76+ |

## Performance

- Lighthouse Score: 95+
- First Contentful Paint: < 1.5s
- Time to Interactive: < 3s
- Cumulative Layout Shift: < 0.1

## SEO

- Meta tags optimized
- Open Graph tags
- Twitter Card tags
- Structured data (JSON-LD)
- Sitemap.xml
- robots.txt

## Analytics

- Google Analytics 4
- Mixpanel
- Sentry (error tracking)
- Hotjar (heatmaps)

## Security Headers

```nginx
add_header X-Frame-Options "SAMEORIGIN";
add_header X-Content-Type-Options "nosniff";
add_header X-XSS-Protection "1; mode=block";
add_header Referrer-Policy "strict-origin-when-cross-origin";
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';";
```

## API Integration

### REST API

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8097';

export const fetchAPI = async <T>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = localStorage.getItem('tigerwallet-token');
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  return response.json();
};
```

### WebSocket

```typescript
const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8095';

export const createWebSocket = () => {
  const ws = new WebSocket(WS_URL);
  
  ws.onopen = () => console.log('Connected to WebSocket');
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    handleMessage(data);
  };
  ws.onclose = () => console.log('Disconnected from WebSocket');
  
  return ws;
};
```
