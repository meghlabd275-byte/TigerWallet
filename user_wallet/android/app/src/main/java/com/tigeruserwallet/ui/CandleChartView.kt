package com.tigeruserwallet.ui

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.util.AttributeSet
import android.view.View

/**
 * Real OHLC candlestick chart (Open-High-Low-Close). Renders whatever candles
 * the caller supplies from the backend /terminal/kline endpoint. Zero
 * dependencies, real canvas draw — no libraries, no fabricated data.
 */
class CandleChartView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null
) : View(context, attrs) {

    data class Candle(val open: Float, val high: Float, val low: Float, val close: Float)

    private val upPaint = Paint().apply { color = Color.parseColor("#16a34a") }
    private val downPaint = Paint().apply { color = Color.parseColor("#dc2626") }
    private val textPaint = Paint().apply {
        color = Color.GRAY
        textSize = 28f
        isAntiAlias = true
    }

    private var candles: List<Candle> = emptyList()

    fun setCandles(list: List<Candle>) {
        candles = list
        invalidate()
    }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        if (candles.isEmpty()) {
            canvas.drawText("No candle data for this symbol/range.", 16f, 48f, textPaint)
            return
        }
        val padX = 80f
        var min = Float.MAX_VALUE
        var max = -Float.MAX_VALUE
        for (c in candles) {
            if (c.low < min) min = c.low
            if (c.high > max) max = c.high
        }
        val span = if (max - min > 0) max - min else 1f
        val bw = (width - padX) / candles.size - 4f
        fun y(v: Float): Float = height - ((v - min) / span) * (height - 48f) - 24f
        candles.forEachIndexed { i, c ->
            val up = c.close >= c.open
            val paint = if (up) upPaint else downPaint
            val x = padX + i * (bw + 4f)
            // Wick
            canvas.drawLine(x + bw / 2f, y(c.high), x + bw / 2f, y(c.low), paint)
            // Body
            val top = y(maxOf(c.open, c.close))
            val bottom = y(minOf(c.open, c.close))
            val height = (bottom - top).coerceAtLeast(2f)
            canvas.drawRect(x, top, x + bw, top + height, paint)
        }
        canvas.drawText("%.2f".format(max), 8f, y(max) + 8f, textPaint)
        canvas.drawText("%.2f".format(min), 8f, y(min) + 24f, textPaint)
    }
}
