# TigerWallet SDK - Go

Official Go SDK for TigerWallet White Label Integration

## Installation

```bash
go get github.com/tigerwallet/sdk/go
```

## Quick Start

```go
package main

import (
    "github.com/tigerwallet/sdk/go"
)

func main() {
    // Create client
    client := tigerwallet.NewClient(
        "your-api-key",
        "https://api.tigerwallet.com",
        "your-tenant-id",
    )
    
    // Authenticate
    auth := tigerwallet.NewAuthService(client)
    resp, err := auth.Login("user@example.com", "password")
    if err != nil {
        panic(err)
    }
    
    // Use services
    wallet := tigerwallet.NewWalletService(client)
    wallets, err := wallet.ListWallets()
}
```

## Services

### Authentication

```go
auth := tigerwallet.NewAuthService(client)

// Login
resp, err := auth.Login(email, password)

// Register
resp, err := auth.Register(email, password, name)

// Refresh token
newToken, err := auth.RefreshToken(oldToken)
```

### Wallets

```go
wallet := tigerwallet.NewWalletService(client)

// Create wallet
w, err := wallet.CreateWallet(tigerwallet.CreateWalletRequest{
    Chain: "ethereum",
    Type:  "eoa",
    Name: "My Wallet",
})

// Get wallet
w, err := wallet.GetWallet(walletID)

// List wallets
wallets, err := wallet.ListWallets()

// Send transaction
tx, err := wallet.SendTransaction(tigerwallet.TransactionRequest{
    From:   "0xfrom...",
    To:     "0xto...",
    Amount: "1.0",
    Chain:  "ethereum",
    Token:  "ETH",
})
```

### Billing

```go
billing := tigerwallet.NewBillingService(client)

// Get plans
plans, err := billing.GetPlans()

// Get subscription
sub, err := billing.GetSubscription()

// Upgrade
sub, err := billing.UpgradeSubscription(tigerwallet.UpgradeRequest{
    PlanID:       "plan-id",
    BillingCycle: "monthly",
})

// Get usage
usage, err := billing.GetUsage()

// Get invoices
invoices, err := billing.GetInvoices()
```

### Data Fetching

```go
fetcher := tigerwallet.NewFetcherService(client)

// Get prices
prices, err := fetcher.GetPrice([]string{"BTC", "ETH", "USDT"})

// Get blockchain data
data, err := fetcher.GetBlockchainData("ethereum")

// Get wallet balance
balance, err := fetcher.GetWalletBalance("ethereum", "0xaddress...")

// Get transactions
txs, err := fetcher.GetTransactions("ethereum", "0xaddress...", 50)
```

### Notifications

```go
notifications := tigerwallet.NewNotificationService(client)

// Get notifications
notifs, err := notifications.GetNotifications()

// Mark as read
err := notifications.MarkAsRead(notificationID)

// Send notification
err = notifications.SendNotification(tigerwallet.SendNotificationRequest{
    Type:    "transaction",
    Title:   "Transaction Confirmed",
    Message: "Your transaction has been confirmed",
    Channel: "email",
})

// Get preferences
prefs, err := notifications.GetPreferences()

// Update preferences
err = notifications.UpdatePreferences(tigerwallet.NotificationPreference{
    EmailEnabled:     true,
    PushEnabled:     true,
    TransactionAlerts: true,
})
```

### KYC Verification

```go
kyc := tigerwallet.NewKYCService(client)

// Get KYC status
status, err := kyc.GetStatus()

// Submit KYC application
appID, err := kyc.Submit(tigerwallet.KYCSubmission{
    FirstName:          "John",
    LastName:           "Doe",
    DateOfBirth:        "1990-01-01",
    Nationality:        "US",
    CountryOfResidence: "US",
    Address:            "123 Main St",
    City:               "New York",
    PostalCode:        "10001",
    PhoneNumber:       "+1234567890",
})

// Upload document
err = kyc.UploadDocument(appID, tigerwallet.DocumentUpload{
    Type:     "id_card",
    Side:     "front",
    FileData: "base64-encoded-image...",
})
```

## Error Handling

```go
import "github.com/tigerwallet/sdk/go/errors"

resp, err := client.doRequest(...)
if err != nil {
    if apiErr, ok := err.(errors.APIError); ok {
        // Handle API error
        fmt.Println("Code:", apiErr.Code)
        fmt.Println("Message:", apiErr.Message)
    }
    // Handle network error
}
```

## License

MIT
