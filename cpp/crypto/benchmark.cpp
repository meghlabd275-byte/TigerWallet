/**
 * TigerCrypto Performance Benchmark
 * Tests ultra-low latency cryptographic operations
 */

#include "tiger_crypto.h"
#include <chrono>
#include <iostream>
#include <iomanip>
#include <vector>
#include <random>

using namespace TigerCrypto;
using namespace std::chrono;

void print_separator() {
    std::cout << "========================================" << std::endl;
}

int main() {
    std::cout << "\n";
    print_separator();
    std::cout << "  TigerCrypto Performance Benchmark" << std::endl;
    print_separator();
    std::cout << "\n";
    
    // Initialize crypto engine
    CryptoEngine crypto;
    auto result = crypto.initialize(true, std::thread::hardware_concurrency());
    
    if (result != CryptoResult::SUCCESS) {
        std::cerr << "Failed to initialize crypto engine: " << static_cast<int>(result) << std::endl;
        return 1;
    }
    
    std::cout << "Initializing with " << std::thread::hardware_concurrency() << " threads\n" << std::endl;
    
    // Benchmark 1: Key Generation
    print_separator();
    std::cout << "  1. Key Generation (10,000 ops)" << std::endl;
    print_separator();
    
    auto start = high_resolution_clock::now();
    std::vector<KeyPair> keypairs;
    keypairs.reserve(10000);
    
    for (int i = 0; i < 10000; ++i) {
        keypairs.push_back(crypto.generate_keypair());
    }
    
    auto end = high_resolution_clock::now();
    auto duration = duration_cast<microseconds>(end - start);
    
    std::cout << "  Total time:    " << duration.count() / 1000.0 << " ms" << std::endl;
    std::cout << "  Per operation: " << duration.count() / 10000.0 << " µs" << std::endl;
    std::cout << "  Ops/second:    " << std::fixed << std::setprecision(0) 
              << (10000000.0 / duration.count()) << "\n" << std::endl;
    
    // Benchmark 2: Signing
    print_separator();
    std::cout << "  2. ECDSA Signing (100,000 ops)" << std::endl;
    print_separator();
    
    std::string test_message = "TigerWallet - High Performance Crypto";
    std::vector<uint8_t> message(test_message.begin(), test_message.end());
    
    start = high_resolution_clock::now();
    std::vector<Signature> signatures;
    signatures.reserve(100000);
    
    for (int i = 0; i < 100000; ++i) {
        Signature sig;
        crypto.sign(keypairs[0].private_key, message.data(), message.size(), sig);
        signatures.push_back(sig);
    }
    
    end = high_resolution_clock::now();
    duration = duration_cast<microseconds>(end - start);
    
    std::cout << "  Total time:    " << duration.count() / 1000.0 << " ms" << std::endl;
    std::cout << "  Per operation: " << duration.count() / 100000.0 << " µs" << std::endl;
    std::cout << "  Ops/second:    " << std::fixed << std::setprecision(0) 
              << (100000000.0 / duration.count()) << "\n" << std::endl;
    
    // Benchmark 3: Verification
    print_separator();
    std::cout << "  3. ECDSA Verification (100,000 ops)" << std::endl;
    print_separator();
    
    start = high_resolution_clock::now();
    int verified = 0;
    
    for (int i = 0; i < 100000; ++i) {
        auto res = crypto.verify(keypairs[0].public_key, message.data(), 
                                message.size(), signatures[i]);
        if (res == CryptoResult::SUCCESS) verified++;
    }
    
    end = high_resolution_clock::now();
    duration = duration_cast<microseconds>(end - start);
    
    std::cout << "  Total time:    " << duration.count() / 1000.0 << " ms" << std::endl;
    std::cout << "  Per operation: " << duration.count() / 100000.0 << " µs" << std::endl;
    std::cout << "  Ops/second:    " << std::fixed << std::setprecision(0) 
              << (100000000.0 / duration.count()) << std::endl;
    std::cout << "  Verified:      " << verified << " / 100000\n" << std::endl;
    
    // Benchmark 4: AES-256-GCM Encryption
    print_separator();
    std::cout << "  4. AES-256-GCM Encryption (50,000 ops)" << std::endl;
    print_separator();
    
    std::vector<uint8_t> plaintext(1024); // 1KB
    std::random_device rd;
    std::array<uint8_t, 32> key;
    for (auto& b : key) b = rd();
    
    start = high_resolution_clock::now();
    std::vector<EncryptedData> encrypted;
    encrypted.reserve(50000);
    
    for (int i = 0; i < 50000; ++i) {
        EncryptedData enc;
        crypto.encrypt_aes256gcm(plaintext.data(), plaintext.size(), key, enc);
        encrypted.push_back(enc);
    }
    
    end = high_resolution_clock::now();
    duration = duration_cast<microseconds>(end - start);
    
    std::cout << "  Total time:    " << duration.count() / 1000.0 << " ms" << std::endl;
    std::cout << "  Per operation: " << duration.count() / 50000.0 << " µs" << std::endl;
    std::cout << "  Ops/second:    " << std::fixed << std::setprecision(0) 
              << (50000000.0 / duration.count()) << "\n" << std::endl;
    
    // Benchmark 5: Mnemonic Generation
    print_separator();
    std::cout << "  5. BIP-39 Mnemonic Generation (10,000 ops)" << std::endl;
    print_separator();
    
    start = high_resolution_clock::now();
    std::vector<std::string> mnemonics;
    mnemonics.reserve(10000);
    
    for (int i = 0; i < 10000; ++i) {
        mnemonics.push_back(crypto.generate_mnemonic());
    }
    
    end = high_resolution_clock::now();
    duration = duration_cast<microseconds>(end - start);
    
    std::cout << "  Total time:    " << duration.count() / 1000.0 << " ms" << std::endl;
    std::cout << "  Per operation: " << duration.count() / 10000.0 << " µs" << std::endl;
    std::cout << "  Ops/second:    " << std::fixed << std::setprecision(0) 
              << (10000000.0 / duration.count()) << "\n" << std::endl;
    
    // Print sample keypair
    print_separator();
    std::cout << "  Sample Generated Data" << std::endl;
    print_separator();
    std::cout << "\n  Mnemonic: " << mnemonics[0].substr(0, 60) << "..." << std::endl;
    std::cout << "  Address:   " << keypairs[0].address << std::endl;
    std::cout << "  Valid:     " << (keypairs[0].is_valid() ? "YES" : "NO") << std::endl;
    
    // Get engine metrics
    auto metrics = crypto.get_metrics();
    print_separator();
    std::cout << "  Engine Metrics" << std::endl;
    print_separator();
    std::cout << "\n  Operations completed: " << metrics.operations_completed.load() << std::endl;
    std::cout << "  Average latency:      " << std::fixed << std::setprecision(2) 
              << metrics.average_latency_ns() << " ns" << std::endl;
    
    std::cout << "\n" << print_separator << std::endl;
    std::cout << "  Benchmark Complete!" << std::endl;
    print_separator();
    std::cout << "\n";
    
    return 0;
}
