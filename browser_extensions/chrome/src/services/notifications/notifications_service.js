/**
 * TigerWallet Chrome Extension - Real-Time Notifications Service
 * Production-ready push notifications and alerts
 */

class NotificationsService {
    constructor() {
        this.notifications = new Map();
        this.preferences = {
            email: { enabled: true, address: '' },
            push: { enabled: true },
            inApp: { enabled: true },
            types: {
                transaction: { enabled: true, critical: true },
                price: { enabled: true, critical: false },
                security: { enabled: true, critical: true },
                system: { enabled: true, critical: true },
                wallet: { enabled: true, critical: false }
            }
        };
        this.initialized = false;
    }

    async initialize() {
        if (this.initialized) return true;
        await this.loadNotifications();
        await this.loadPreferences();
        this.startPriceMonitoring();
        this.initialized = true;
        console.log('[Notifications] Service initialized');
        return true;
    }

    async loadNotifications() {
        try {
            const result = await chrome.storage.local.get('notifications');
            if (result.notifications) {
                for (const notif of result.notifications) {
                    this.notifications.set(notif.id, notif);
                }
            }
        } catch (error) {
            console.error('[Notifications] Load failed:', error);
        }
    }

    async loadPreferences() {
        try {
            const result = await chrome.storage.local.get('notification_preferences');
            if (result.notification_preferences) {
                this.preferences = result.notification_preferences;
            }
        } catch (error) {
            console.error('[Notifications] Load preferences failed:', error);
        }
    }

    async saveNotifications() {
        try {
            const notifications = Array.from(this.notifications.values());
            await chrome.storage.local.set({ notifications });
        } catch (error) {
            console.error('[Notifications] Save failed:', error);
        }
    }

    async savePreferences() {
        try {
            await chrome.storage.local.set({ notification_preferences: this.preferences });
        } catch (error) {
            console.error('[Notifications] Save preferences failed:', error);
        }
    }

    /**
     * Create and send notification
     */
    async notify(type, title, message, data = {}) {
        const notif = {
            id: this.generateId(),
            type,
            title,
            message,
            data,
            status: 'unread',
            createdAt: Date.now(),
            deliveredAt: null
        };

        this.notifications.set(notif.id, notif);
        await this.saveNotifications();

        // Check if type is enabled
        if (!this.preferences.types[type]?.enabled) {
            return notif;
        }

        // Send via enabled channels
        if (this.preferences.inApp.enabled) {
            this.showInAppNotification(notif);
        }

        if (this.preferences.push.enabled) {
            this.sendPushNotification(notif);
        }

        if (this.preferences.email.enabled && this.preferences.email.address) {
            this.sendEmailNotification(notif);
        }

        // Update status
        notif.deliveredAt = Date.now();
        await this.saveNotifications();

        return notif;
    }

    /**
     * Show in-app notification
     */
    showInAppNotification(notif) {
        // Use Chrome notifications API
        if (chrome.notifications) {
            chrome.notifications.create(notif.id, {
                type: 'basic',
                iconUrl: 'icons/icon-128.png',
                title: notif.title,
                message: notif.message,
                priority: this.preferences.types[notif.type]?.critical ? 1 : 0
            }, (notificationId) => {
                if (chrome.runtime.lastError) {
                    console.error('[Notifications] Show failed:', chrome.runtime.lastError);
                }
            });
        }

        // Also show in popup if open
        chrome.runtime.sendMessage({
            action: 'show_notification',
            notification: notif
        }).catch(() => {});
    }

    /**
     * Send push notification
     */
    async sendPushNotification(notif) {
        // In production, use FCM or similar
        console.log('[Notifications] Push:', notif.title, notif.message);
    }

    /**
     * Send email notification
     */
    async sendEmailNotification(notif) {
        // In production, call backend email service
        console.log('[Notifications] Email to', this.preferences.email.address, notif.title);
    }

    /**
     * Get all notifications
     */
    getAllNotifications(limit = 50) {
        return Array.from(this.notifications.values())
            .sort((a, b) => b.createdAt - a.createdAt)
            .slice(0, limit);
    }

    /**
     * Get unread notifications
     */
    getUnreadNotifications() {
        return Array.from(this.notifications.values())
            .filter(n => n.status === 'unread')
            .sort((a, b) => b.createdAt - a.createdAt);
    }

    /**
     * Get unread count
     */
    getUnreadCount() {
        return Array.from(this.notifications.values())
            .filter(n => n.status === 'unread').length;
    }

    /**
     * Mark notification as read
     */
    async markAsRead(notificationId) {
        const notif = this.notifications.get(notificationId);
        if (notif) {
            notif.status = 'read';
            await this.saveNotifications();
        }
        return notif;
    }

    /**
     * Mark all as read
     */
    async markAllAsRead() {
        for (const notif of this.notifications.values()) {
            if (notif.status === 'unread') {
                notif.status = 'read';
            }
        }
        await this.saveNotifications();
    }

    /**
     * Delete notification
     */
    async deleteNotification(notificationId) {
        if (this.notifications.has(notificationId)) {
            this.notifications.delete(notificationId);
            await this.saveNotifications();
            return true;
        }
        return false;
    }

    /**
     * Clear all notifications
     */
    async clearAll() {
        this.notifications.clear();
        await this.saveNotifications();
    }

    /**
     * Get notification preferences
     */
    getPreferences() {
        return { ...this.preferences };
    }

    /**
     * Update notification preferences
     */
    async updatePreferences(newPreferences) {
        this.preferences = { ...this.preferences, ...newPreferences };
        await this.savePreferences();
    }

    /**
     * Transaction notification
     */
    async notifyTransaction(txHash, status, amount, symbol, isIncoming) {
        return this.notify('transaction', 
            isIncoming ? 'Incoming Transaction' : 'Outgoing Transaction',
            `${isIncoming ? 'Received' : 'Sent'} ${amount} ${symbol}`,
            { txHash, status, amount, symbol, isIncoming }
        );
    }

    /**
     * Price alert notification
     */
    async notifyPriceAlert(symbol, currentPrice, targetPrice, condition) {
        return this.notify('price',
            'Price Alert',
            `${symbol} is now ${condition} ${targetPrice} (Current: ${currentPrice})`,
            { symbol, currentPrice, targetPrice, condition }
        );
    }

    /**
     * Security alert notification
     */
    async notifySecurityAlert(alertType, description, details = {}) {
        return this.notify('security',
            'Security Alert',
            description,
            { alertType, ...details }
        );
    }

    /**
     * System notification
     */
    async notifySystem(title, message) {
        return this.notify('system', title, message);
    }

    /**
     * Start price monitoring
     */
    startPriceMonitoring() {
        this.priceMonitorInterval = setInterval(async () => {
            await this.checkPriceAlerts();
        }, 60000); // Check every minute
    }

    /**
     * Stop price monitoring
     */
    stopPriceMonitoring() {
        if (this.priceMonitorInterval) {
            clearInterval(this.priceMonitorInterval);
        }
    }

    /**
     * Check price alerts
     */
    async checkPriceAlerts() {
        if (!this.preferences.types.price?.enabled) return;

        try {
            // Get stored price alerts
            const result = await chrome.storage.local.get('price_alerts');
            const alerts = result.price_alerts || [];

            for (const alert of alerts) {
                if (!alert.isActive) continue;

                // Fetch current price (in production, use real API)
                const currentPrice = await this.fetchPrice(alert.symbol);

                if (!currentPrice) continue;

                let triggered = false;
                switch (alert.condition) {
                    case 'above':
                        triggered = parseFloat(currentPrice) > parseFloat(alert.targetPrice);
                        break;
                    case 'below':
                        triggered = parseFloat(currentPrice) < parseFloat(alert.targetPrice);
                        break;
                    case 'cross':
                        triggered = (
                            (parseFloat(currentPrice) > parseFloat(alert.targetPrice) && 
                             parseFloat(alert.lastPrice) <= parseFloat(alert.targetPrice)) ||
                            (parseFloat(currentPrice) < parseFloat(alert.targetPrice) && 
                             parseFloat(alert.lastPrice) >= parseFloat(alert.targetPrice))
                        );
                        break;
                }

                if (triggered) {
                    await this.notifyPriceAlert(
                        alert.symbol,
                        currentPrice,
                        alert.targetPrice,
                        alert.condition
                    );

                    // Update alert
                    alert.lastPrice = currentPrice;
                    if (!alert.repeat) {
                        alert.isActive = false;
                    }
                } else {
                    alert.lastPrice = currentPrice;
                }
            }

            await chrome.storage.local.set({ price_alerts: alerts });
        } catch (error) {
            console.error('[Notifications] Price check failed:', error);
        }
    }

    /**
     * Fetch price (placeholder - use real API in production)
     */
    async fetchPrice(symbol) {
        // In production, call price API
        return null;
    }

    /**
     * Create price alert
     */
    async createPriceAlert(symbol, targetPrice, condition = 'above', repeat = false) {
        const alert = {
            id: this.generateId(),
            symbol,
            targetPrice,
            condition,
            repeat,
            isActive: true,
            lastPrice: null,
            createdAt: Date.now()
        };

        try {
            const result = await chrome.storage.local.get('price_alerts');
            const alerts = result.price_alerts || [];
            alerts.push(alert);
            await chrome.storage.local.set({ price_alerts: alerts });
        } catch (error) {
            console.error('[Notifications] Create price alert failed:', error);
        }

        return alert;
    }

    /**
     * Cancel price alert
     */
    async cancelPriceAlert(alertId) {
        try {
            const result = await chrome.storage.local.get('price_alerts');
            const alerts = result.price_alerts || [];
            const index = alerts.findIndex(a => a.id === alertId);
            if (index !== -1) {
                alerts.splice(index, 1);
                await chrome.storage.local.set({ price_alerts: alerts });
                return true;
            }
        } catch (error) {
            console.error('[Notifications] Cancel price alert failed:', error);
        }
        return false;
    }

    /**
     * Get price alerts
     */
    async getPriceAlerts() {
        try {
            const result = await chrome.storage.local.get('price_alerts');
            return result.price_alerts || [];
        } catch (error) {
            return [];
        }
    }

    generateId() {
        return 'notif_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }
}

window.NotificationsService = new NotificationsService();
