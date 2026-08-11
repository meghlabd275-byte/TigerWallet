"""
TigerWallet Python SDK
A client library for interacting with TigerWallet API
"""

import hashlib
import hmac
import time
import requests
from typing import List, Optional, Dict, Any


class TigerWalletClient:
    """TigerWallet API Client"""
    
    def __init__(
        self,
        api_key: str,
        api_secret: str,
        base_url: str = "http://localhost:8443",
        tenant_id: Optional[str] = None,
        timeout: int = 30
    ):
        self.api_key = api_key
        self.api_secret = api_secret
        self.base_url = base_url
        self.tenant_id = tenant_id
        self.timeout = timeout
        self.session = requests.Session()
    
    def _generate_signature(
        self,
        method: str,
        path: str,
        timestamp: str,
        body: bytes = b""
    ) -> str:
        """Generate HMAC-SHA256 signature"""
        message = f"{method}\n{path}\n{timestamp}\n{body.decode() if body else ''}"
        signature = hmac.new(
            self.api_secret.encode(),
            message.encode(),
            hashlib.sha256
        ).hexdigest()
        return signature
    
    def _request(
        self,
        method: str,
        path: str,
        data: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """Make authenticated API request"""
        timestamp = str(int(time.time()))
        
        body = b""
        if data:
            import json
            body = json.dumps(data).encode()
        
        signature = self._generate_signature(method, path, timestamp, body)
        
        url = f"{self.base_url}{path}"
        headers = {
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
            "X-Timestamp": timestamp,
            "X-Signature": signature
        }
        
        if self.tenant_id:
            headers["X-Tenant-ID"] = self.tenant_id
        
        response = self.session.request(
            method=method,
            url=url,
            data=body,
            headers=headers,
            timeout=self.timeout
        )
        
        response.raise_for_status()
        return response.json()
    
    # Fetcher Service
    def get_prices(self, symbols: List[str]) -> Dict[str, Any]:
        """Get token prices"""
        path = f"/api/v1/fetcher/prices?symbols={','.join(symbols)}"
        return self._request("GET", path)
    
    def get_wallet_balance(self, chain: str, address: str) -> Dict[str, Any]:
        """Get wallet balance"""
        path = f"/api/v1/fetcher/wallet/{chain}/{address}"
        return self._request("GET", path)
    
    def get_transactions(
        self,
        chain: str,
        address: str,
        limit: int = 50
    ) -> Dict[str, Any]:
        """Get transactions"""
        path = f"/api/v1/fetcher/transactions/{chain}/{address}?limit={limit}"
        return self._request("GET", path)
    
    def get_token_info(self, chain: str, token_address: str) -> Dict[str, Any]:
        """Get token information"""
        path = f"/api/v1/fetcher/token/{chain}/{token_address}"
        return self._request("GET", path)
    
    def get_market_data(self, symbols: List[str]) -> Dict[str, Any]:
        """Get market data"""
        path = f"/api/v1/fetcher/market?symbols={','.join(symbols)}"
        return self._request("GET", path)
    
    # Permission Service
    def get_permissions(self) -> Dict[str, Any]:
        """Get all permissions"""
        return self._request("GET", "/api/v1/permissions")
    
    def check_permission(self, feature: str) -> bool:
        """Check if a feature is enabled"""
        path = f"/api/v1/permissions/{feature}"
        result = self._request("GET", path)
        return result.get("enabled", False)
    
    def sync_permissions(self) -> Dict[str, Any]:
        """Sync permissions from server"""
        return self._request("POST", "/api/v1/permissions/sync")
    
    # License Service
    def validate_license(self, license_key: str, hardware_id: str) -> Dict[str, Any]:
        """Validate license key"""
        return self._request("POST", "/api/v1/licenses/validate", {
            "license_key": license_key,
            "hardware_id": hardware_id
        })
    
    def get_license_info(self) -> Dict[str, Any]:
        """Get license information"""
        return self._request("GET", "/api/v1/licenses/info")
    
    # Webhook Service
    def register_webhook(
        self,
        event_type: str,
        url: str,
        secret: str
    ) -> Dict[str, Any]:
        """Register a webhook"""
        return self._request("POST", "/api/v1/webhooks", {
            "event_type": event_type,
            "url": url,
            "secret": secret
        })
    
    def verify_webhook(self, payload: str, signature: str, secret: str) -> bool:
        """Verify webhook signature"""
        expected = hmac.new(
            secret.encode(),
            payload.encode(),
            hashlib.sha256
        ).hexdigest()
        return hmac.compare_digest(signature, expected)


class FetcherService:
    """Fetcher service for blockchain data"""
    
    def __init__(self, client: TigerWalletClient):
        self.client = client
    
    def get_prices(self, symbols: List[str]) -> Dict[str, Any]:
        return self.client.get_prices(symbols)
    
    def get_wallet_balance(self, chain: str, address: str) -> Dict[str, Any]:
        return self.client.get_wallet_balance(chain, address)
    
    def get_transactions(
        self,
        chain: str,
        address: str,
        limit: int = 50
    ) -> Dict[str, Any]:
        return self.client.get_transactions(chain, address, limit)
    
    def get_token_info(self, chain: str, token_address: str) -> Dict[str, Any]:
        return self.client.get_token_info(chain, token_address)
    
    def get_market_data(self, symbols: List[str]) -> Dict[str, Any]:
        return self.client.get_market_data(symbols)


class PermissionService:
    """Permission service"""
    
    def __init__(self, client: TigerWalletClient):
        self.client = client
    
    def get_permissions(self) -> Dict[str, Any]:
        return self.client.get_permissions()
    
    def check_permission(self, feature: str) -> bool:
        return self.client.check_permission(feature)
    
    def sync_permissions(self) -> Dict[str, Any]:
        return self.client.sync_permissions()


class LicenseService:
    """License service"""
    
    def __init__(self, client: TigerWalletClient):
        self.client = client
    
    def validate(self, license_key: str, hardware_id: str) -> Dict[str, Any]:
        return self.client.validate_license(license_key, hardware_id)
    
    def get_info(self) -> Dict[str, Any]:
        return self.client.get_license_info()


class WebhookService:
    """Webhook service"""
    
    def __init__(self, client: TigerWalletClient):
        self.client = client
    
    def register(
        self,
        event_type: str,
        url: str,
        secret: str
    ) -> Dict[str, Any]:
        return self.client.register_webhook(event_type, url, secret)
    
    def verify(self, payload: str, signature: str, secret: str) -> bool:
        return self.client.verify_webhook(payload, signature, secret)
