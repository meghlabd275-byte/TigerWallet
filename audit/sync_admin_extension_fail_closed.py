from pathlib import Path

files = [
    Path('admin/extensions/firefox/js/popup.js'),
    Path('admin/extensions/safari/js/popup.js'),
]
replacements = [
    ("""        } catch (error):\n""", """        } catch (error):\n"""),
]

for path in files:
    text = path.read_text()
    pairs = [
        ("""        } catch (error) {\n            console.error('Load KYC error:', error);\n            updateKYCCounts({ pending: 15, approved: 120, rejected: 5 });\n            renderKYC(generateDemoKYC());\n        }\n""", """        } catch (error) {\n            console.error('Load KYC error:', error);\n            updateKYCCounts({});\n            elements.kycTableBody.innerHTML = '<tr><td colspan=\"5\" class=\"empty\">KYC data is unavailable. Reconnect to the TigerWallet admin API and retry.</td></tr>';\n        }\n"""),
        ("""    function generateDemoKYC() {\n        const statuses = ['pending', 'approved', 'rejected'];\n        \n        return Array.from({ length: 10 }, (_, i) => ({\n            id: `kyc_${i + 1}`,\n            email: `user${i + 1}@example.com`,\n            level: i % 3 + 1,\n            status: statuses[i % statuses.length],\n            submittedAt: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString()\n        }));\n    }\n\n""", ""),
        ("""        } catch (error) {\n            console.error('Load tokens error:', error);\n            renderTokens(generateDemoTokens());\n        }\n""", """        } catch (error) {\n            console.error('Load tokens error:', error);\n            elements.tokensGrid.innerHTML = '<div class=\"empty\">Token data is unavailable. Reconnect to the TigerWallet admin API and retry.</div>';\n        }\n"""),
        ("""    function generateDemoTokens() {\n        return [\n            { id: '1', symbol: 'ETH', name: 'Ethereum', logo: '', price: 3250.00, marketCap: 390000000000, volume24h: 15000000000, isActive: true },\n            { id: '2', symbol: 'USDT', name: 'Tether', logo: '', price: 1.00, marketCap: 95000000000, volume24h: 50000000000, isActive: true },\n            { id: '3', symbol: 'BTC', name: 'Bitcoin', logo: '', price: 67500.00, marketCap: 1320000000000, volume24h: 35000000000, isActive: true },\n            { id: '4', symbol: 'BNB', name: 'BNB', logo: '', price: 580.00, marketCap: 87000000000, volume24h: 1800000000, isActive: true },\n            { id: '5', symbol: 'SOL', name: 'Solana', logo: '', price: 145.00, marketCap: 64000000000, volume24h: 2500000000, isActive: true },\n            { id: '6', symbol: 'XRP', name: 'Ripple', logo: '', price: 0.52, marketCap: 28000000000, volume24h: 1500000000, isActive: true }\n        ];\n    }\n\n""", ""),
        ("""        } catch (error) {\n            console.error('Load withdrawals error:', error);\n            renderWithdrawals(generateDemoWithdrawals());\n        }\n""", """        } catch (error) {\n            console.error('Load withdrawals error:', error);\n            elements.withdrawalsTableBody.innerHTML = '<tr><td colspan=\"6\" class=\"empty\">Withdrawal data is unavailable. Reconnect to the TigerWallet admin API and retry.</td></tr>';\n        }\n"""),
        ("""    function generateDemoWithdrawals() {\n        const statuses = ['pending', 'approved', 'rejected', 'processing', 'completed'];\n        const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'solana'];\n        \n        return Array.from({ length: 10 }, (_, i) => ({\n            id: `wd_${i + 1}`,\n            userEmail: `user${i + 1}@example.com`,\n            amount: (Math.random() * 10000).toFixed(2),\n            token: 'USDT',\n            chain: chains[i % chains.length],\n            address: `0x${Math.random().toString(16).substring(2, 42)}`,\n            status: statuses[i % statuses.length],\n            createdAt: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString()\n        }));\n    }\n\n""", ""),
    ]
    for old, new in pairs:
        if old not in text:
            raise SystemExit(f'missing expected block in {path}: {old[:70]!r}')
        text = text.replace(old, new, 1)
    path.write_text(text)
print('synchronized', ', '.join(str(p) for p in files))
