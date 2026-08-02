/**
 * TigerWallet Desktop - Main Entry Point
 * Testing all wallet services
 */

#include <iostream>
#include <memory>
#include <thread>
#include <chrono>

#include "models/wallet_models.h"
#include "services/blockchain_service.h"
#include "services/price_service.h"
#include "services/swap_service.h"
#include "services/staking_service.h"
#include "services/nft_service.h"
#include "services/api_client.h"
#include "services/keychain_manager.h"

using namespace tiger::wallet;

void printHeader(const std::string& title) {
    std::cout << "\n" << std::string(60, '=') << "\n";
    std::cout << "  " << title << "\n";
    std::cout << std::string(60, '=') << "\n";
}

void testBlockchainService() {
    printHeader("Testing BlockchainService");
    
    auto service = BlockchainService::getInstance();
    service->initialize();
    
    // Get supported chains
    auto chains = service->getSupportedChains();
    std::cout << "Supported chains: " << chains.size() << "\n";
    for (const auto& chain : chains) {
        std::cout << "  - " << chain.name << " (" << chain.symbol << ")\n";
    }
    
    // Create wallet test
    auto chain = service->getChain("ethereum").value();
    auto walletFuture = service->createWallet(chain, "Test Wallet");
    auto wallet = walletFuture.get();
    std::cout << "\nCreated wallet:\n";
    std::cout << "  ID: " << wallet.id << "\n";
    std::cout << "  Address: " << wallet.address << "\n";
    std::cout << "  Chain: " << wallet.chain_id << "\n";
    
    service->shutdown();
}

void testPriceService() {
    printHeader("Testing PriceService");
    
    auto service = PriceService::getInstance();
    service->initialize();
    
    // Get Bitcoin price
    auto priceFuture = service->getPrice("BTC");
    auto price = priceFuture.get();
    std::cout << "Bitcoin Price:\n";
    std::cout << "  Price: " << price.getFormattedPrice() << "\n";
    std::cout << "  Change 24h: " << price.getFormattedChange() << "\n";
    std::cout << "  Market Cap: " << price.getFormattedMarketCap() << "\n";
    
    // Get multiple prices
    std::vector<std::string> symbols = {"ETH", "SOL", "MATIC"};
    auto pricesFuture = service->getPrices(symbols);
    auto prices = pricesFuture.get();
    std::cout << "\nMultiple Prices:\n";
    for (const auto& pair : prices) {
        std::cout << "  " << pair.first << ": " << pair.second.getFormattedPrice() << "\n";
    }
    
    service->shutdown();
}

void testSwapService() {
    printHeader("Testing SwapService");
    
    auto service = SwapService::getInstance();
    service->initialize();
    
    // Get swap quote
    auto quoteFuture = service->getQuote("ETH", "USDC", "1.0", "ethereum");
    auto quote = quoteFuture.get();
    std::cout << "Swap Quote:\n";
    std::cout << "  From: " << quote.getDisplayFromAmount() << "\n";
    std::cout << "  To: " << quote.getDisplayToAmount() << "\n";
    std::cout << "  Price Impact: " << quote.price_impact << "%\n";
    std::cout << "  Gas Estimate: " << quote.gas_estimate << " ETH\n";
    
    service->shutdown();
}

void testStakingService() {
    printHeader("Testing StakingService");
    
    auto service = StakingService::getInstance();
    service->initialize();
    
    // Get staking quote
    auto quoteFuture = service->getStakingQuote("ethereum", "ETH");
    auto quote = quoteFuture.get();
    std::cout << "Staking Quote:\n";
    std::cout << "  APY: " << quote.apy << "%\n";
    std::cout << "  Min Stake: " << quote.min_stake << " ETH\n";
    std::cout << "  Lock Period: " << quote.lock_period_days << " days\n";
    
    // Get validators
    auto validatorsFuture = service->getValidators("ethereum");
    auto validators = validatorsFuture.get();
    std::cout << "\nValidators: " << validators.size() << " available\n";
    
    service->shutdown();
}

void testNFTService() {
    printHeader("Testing NFTService");
    
    auto service = NFTService::getInstance();
    service->initialize();
    
    // Get collection info
    auto infoFuture = service->getCollectionInfo("0x1234...", "ethereum");
    auto info = infoFuture.get();
    std::cout << "Collection Info:\n";
    std::cout << "  Name: " << info.name << "\n";
    std::cout << "  Symbol: " << info.symbol << "\n";
    std::cout << "  Total Supply: " << info.total_supply << "\n";
    
    service->shutdown();
}

void testKeychainManager() {
    printHeader("Testing KeychainManager");
    
    auto keychain = KeychainManager::getInstance();
    keychain->initialize();
    
    // Save and load test
    std::string testMnemonic = "abandon ability able about above absent absorb abstract absurd abuse access accident";
    bool saved = keychain->saveWalletSeed("test-wallet-1", testMnemonic);
    std::cout << "Saved mnemonic: " << (saved ? "SUCCESS" : "FAILED") << "\n";
    
    auto loaded = keychain->loadWalletSeed("test-wallet-1");
    if (loaded) {
        std::cout << "Loaded mnemonic: " << loaded->substr(0, 20) << "...\n";
    }
    
    // Test session token
    keychain->setSessionToken("test-session-token-12345");
    auto session = keychain->getSessionToken();
    if (session) {
        std::cout << "Session token: " << *session << "\n";
    }
    
    keychain->shutdown();
}

void testAPIClient() {
    printHeader("Testing APIClient");
    
    auto client = APIClient::getInstance();
    client->initialize("https://api.tigerwallet.com");
    client->setAuthToken("test-auth-token");
    
    std::cout << "API Client initialized successfully\n";
    std::cout << "Base URL: https://api.tigerwallet.com\n";
    
    client->shutdown();
}

int main() {
    std::cout << "\n";
    std::cout << "╔═══════════════════════════════════════════════════════════╗\n";
    std::cout << "║         TigerWallet Desktop - Service Tests              ║\n";
    std::cout << "║         Complete Wallet Functionality Test               ║\n";
    std::cout << "╚═══════════════════════════════════════════════════════════╝\n";
    
    try {
        // Test all services
        testBlockchainService();
        testPriceService();
        testSwapService();
        testStakingService();
        testNFTService();
        testKeychainManager();
        testAPIClient();
        
        std::cout << "\n" << std::string(60, '=') << "\n";
        std::cout << "  ALL TESTS COMPLETED SUCCESSFULLY!\n";
        std::cout << std::string(60, '=') << "\n\n";
        
    } catch (const std::exception& e) {
        std::cerr << "ERROR: " << e.what() << std::endl;
        return 1;
    }
    
    return 0;
}
