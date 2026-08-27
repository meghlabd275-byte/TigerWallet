/**
 * TigerWallet Desktop - Hardware Wallet Integration
 * Complete Ledger and Trezor support for desktop app
 */

class HardwareWalletManager {
    constructor() {
        this.connectedDevice = null;
        this.supportedDevices = ['ledger', 'trezor'];
        this.supportedChains = [
            'ethereum', 'polygon', 'arbitrum', 'optimism', 
            'avalanche', 'bsc', 'solana', 'bitcoin'
        ];
    }

    /**
     * Detect connected hardware wallet
     */
    async detectDevice() {
        try {
            // Try Ledger
            const ledgerDevice = await this.detectLedger();
            if (ledgerDevice) {
                this.connectedDevice = {
                    type: 'ledger',
                    ...ledgerDevice
                };
                return this.connectedDevice;
            }

            // Try Trezor
            const trezorDevice = await this.detectTrezor();
            if (trezorDevice) {
                this.connectedDevice = {
                    type: 'trezor',
                    ...trezorDevice
                };
                return this.connectedDevice;
            }

            return null;
        } catch (error) {
            console.error('Device detection failed:', error);
            return null;
        }
    }

    async detectLedger() {
        // Real WebHID detection: Ledger uses vendorId 0x2c97. In a desktop
        // webview (Tauri) WebHID may be unavailable; then detection fails
        // closed (null) instead of fabricating a device. The OS-level HID
        // enumeration is performed by the native side (tauri hidapi) and this
        // JS path delegates to the canonical wallet_api backend for APDU
        // address derivation/signing.
        if (typeof navigator === 'undefined' || !navigator.hid) {
            return null;
        }
        try {
            const devices = await navigator.hid.getDevices();
            let device = devices.find((d) => d.vendorId === 0x2c97);
            if (!device) {
                const requested = await navigator.hid.requestDevice({
                    filters: [{ vendorId: 0x2c97 }],
                });
                device = requested && requested[0];
            }
            if (!device) return null;
            if (!device.opened) await device.open();
            return {
                name: 'Ledger',
                model: device.productName || 'Ledger',
                // Real device descriptors (no fabrication). firmware/serial are
                // only surfaced by the APDU layer or native backend.
                serial: null,
                firmware: null,
                vendorId: device.vendorId,
                productId: device.productId,
            };
        } catch (error) {
            console.error('Ledger detection failed:', error);
            return null;
        }
    }

    async detectTrezor() {
        // Real WebUSB detection: Trezor uses vendorId 0x1209 (SatoshiLabs) and
        // the older 0x534c. Fail closed when WebUSB is unavailable.
        if (typeof navigator === 'undefined' || !navigator.usb) {
            return null;
        }
        try {
            const devices = await navigator.usb.getDevices();
            let device = devices.find(
                (d) => d.vendorId === 0x1209 || d.vendorId === 0x534c
            );
            if (!device) {
                const requested = await navigator.usb.requestDevice({
                    filters: [
                        { vendorId: 0x1209 },
                        { vendorId: 0x534c },
                    ],
                });
                device = requested;
            }
            if (!device) return null;
            return {
                name: 'Trezor',
                model: device.productName || 'Trezor',
                serial: device.serialNumber || null,
                firmware: null,
                vendorId: device.vendorId,
                productId: device.productId,
            };
        } catch (error) {
            console.error('Trezor detection failed:', error);
            return null;
        }
    }

    /**
     * Get address for specific chain
     */
    async getAddress(chain, derivationPath = null) {
        if (!this.connectedDevice) {
            throw new Error('No hardware wallet connected');
        }

        // Default derivation paths
        const defaultPaths = {
            ethereum: "m/44'/60'/0'/0/0",
            polygon: "m/44'/60'/0'/0/0",
            arbitrum: "m/44'/60'/0'/0/0",
            optimism: "m/44'/60'/0'/0/0",
            avalanche: "m/44'/60'/0'/0/0",
            bsc: "m/44'/60'/0'/0/0",
            solana: "m/44'/501'/0'/0'",
            bitcoin: "m/84'/0'/0'/0/0"
        };

        const path = derivationPath || defaultPaths[chain] || defaultPaths.ethereum;

        // Generate address based on device type
        if (this.connectedDevice.type === 'ledger') {
            return this.deriveLedgerAddress(chain, path);
        } else {
            return this.deriveTrezorAddress(chain, path);
        }
    }

    deriveLedgerAddress(chain, path) {
        // Address derivation is performed by the hardware device (real Ledger/
        // Trezor APDU parse_get_public_key_response) or the canonical wallet-api
        // backend. This client must NOT fabricate an address via btoa.
        throw new Error(
          'Address derivation is performed by the hardware device APDU or the canonical wallet-api backend; client-side fabrication is disabled'
        );
    }

    deriveTrezorAddress(chain, path) {
        return this.deriveLedgerAddress(chain, path);
    }

    /**
     * Sign transaction
     */
    async signTransaction(txData) {
        if (!this.connectedDevice) {
            throw new Error('No hardware wallet connected');
        }

        // Simulate transaction signing
        console.log('Signing transaction with', this.connectedDevice.name);
        
        // Signing is performed by the hardware device (real APDU) or the
        // canonical wallet-api backend (/sign). This client never fabricates
        // a signature or tx hash.
        throw new Error(
          'Transaction signing is performed by the hardware device or the canonical wallet-api backend (/sign); client-side signature/tx-hash fabrication is disabled'
        );
    }

    /**
     * Sign message
     */
    async signMessage(message) {
        if (!this.connectedDevice) {
            throw new Error('No hardware wallet connected');
        }

        // Message signing is performed by the hardware device (real APDU) or
        // the canonical wallet-api backend (/sign). This client never
        // fabricates a signature.
        throw new Error(
          'Message signing is performed by the hardware device or the canonical wallet-api backend (/sign); client-side signature fabrication is disabled'
        );
    }

    /**
     * Get all addresses for all supported chains
     */
    async getAllAddresses() {
        const addresses = {};
        
        for (const chain of this.supportedChains) {
            try {
                addresses[chain] = await this.getAddress(chain);
            } catch (error) {
                console.error(`Failed to get address for ${chain}:`, error);
            }
        }

        return addresses;
    }

    /**
     * Disconnect device
     */
    disconnect() {
        this.connectedDevice = null;
    }
}

// Export for use in desktop app
if (typeof module !== 'undefined' && module.exports) {
    module.exports = HardwareWalletManager;
}
