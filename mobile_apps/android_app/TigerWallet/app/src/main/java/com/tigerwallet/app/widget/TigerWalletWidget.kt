package com.tigerwallet.app.widget

import android.content.Context
import android.widget.RemoteViews

/**
 * TigerWallet Widget SDK
 * Android App Widget for quick wallet access
 */

class TigerWalletWidget(val context: Context) {

    companion object {
        const val PROVIDER_AUTHORITY = "com.tigerwallet.app.widget"
    }

    enum class WidgetType {
        BALANCE,
        QUICK_SEND,
        PRICE,
        PORTFOLIO,
        NFT
    }

    fun getBalanceWidget(walletAddress: String): RemoteViews {
        return RemoteViews(context.packageName, android.R.layout.simple_list_item_1).apply {
            setTextViewText(android.R.id.text1, "Balance: 0.00 ETH")
        }
    }

    fun getQuickSendWidget(): RemoteViews {
        return RemoteViews(context.packageName, android.R.layout.simple_list_item_1).apply {
            setTextViewText(android.R.id.text1, "Quick Send")
        }
    }

    fun getPriceWidget(token: String): RemoteViews {
        return RemoteViews(context.packageName, android.R.layout.simple_list_item_1).apply {
            setTextViewText(android.R.id.text1, "$token: $0.00")
        }
    }

    fun getPortfolioWidget(walletAddress: String): RemoteViews {
        return RemoteViews(context.packageName, android.R.layout.simple_list_item_1).apply {
            setTextViewText(android.R.id.text1, "Portfolio: $0.00")
        }
    }

    fun getNFTWidget(nftImageUrl: String, nftName: String): RemoteViews {
        return RemoteViews(context.packageName, android.R.layout.simple_list_item_1).apply {
            setTextViewText(android.R.id.text1, nftName)
        }
    }
}

data class WidgetConfig(
    val type: TigerWalletWidget.WidgetType,
    val walletAddress: String? = null,
    val token: String? = null,
    val refreshInterval: Long = 60000,
    val theme: String = "auto"
)

class WidgetManager(private val context: Context) {
    private val widget = TigerWalletWidget(context)

    fun createWidget(config: WidgetConfig): RemoteViews {
        return when (config.type) {
            TigerWalletWidget.WidgetType.BALANCE -> widget.getBalanceWidget(config.walletAddress ?: "")
            TigerWalletWidget.WidgetType.QUICK_SEND -> widget.getQuickSendWidget()
            TigerWalletWidget.WidgetType.PRICE -> widget.getPriceWidget(config.token ?: "ETH")
            TigerWalletWidget.WidgetType.PORTFOLIO -> widget.getPortfolioWidget(config.walletAddress ?: "")
            TigerWalletWidget.WidgetType.NFT -> widget.getNFTWidget("", "NFT")
        }
    }

    fun getAvailableWidgets(): List<WidgetConfig> {
        return listOf(
            WidgetConfig(TigerWalletWidget.WidgetType.BALANCE),
            WidgetConfig(TigerWalletWidget.WidgetType.QUICK_SEND),
            WidgetConfig(TigerWalletWidget.WidgetType.PRICE, token = "BTC"),
            WidgetConfig(TigerWalletWidget.WidgetType.PRICE, token = "ETH"),
            WidgetConfig(TigerWalletWidget.WidgetType.PORTFOLIO),
            WidgetConfig(TigerWalletWidget.WidgetType.NFT)
        )
    }
}
