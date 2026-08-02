/**
 * TigerWallet Full Fetchers - Implementation
 * Ultra Low Latency C++ Implementation
 */

#include "full_fetchers.hpp"
#include <csignal>
#include <iostream>

using namespace tiger;

// Global manager instance
FullFetcherManager* g_manager = nullptr;

void signal_handler(int signal) {
    std::cout << "\nShutting down..." << std::endl;
    if (g_manager) {
        g_manager->stopAll();
    }
    exit(0);
}

int main() {
    std::cout << "TigerWallet Full Fetchers - Ultra Low Latency Engine" << std::endl;
    std::cout << "=====================================================" << std::endl;
    
    // Set up signal handlers
    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);
    
    // Create and initialize manager
    g_manager = new FullFetcherManager();
    
    if (!g_manager->initializeAll()) {
        std::cerr << "Failed to initialize fetchers" << std::endl;
        return 1;
    }
    
    // Start fetchers
    g_manager->startAll();
    
    std::cout << "All fetchers running. Press Ctrl+C to stop." << std::endl;
    
    // Main loop - run for a bit then print stats
    for (int i = 0; i < 10; i++) {
        std::this_thread::sleep_for(std::chrono::seconds(2));
        g_manager->printStats();
    }
    
    // Cleanup
    g_manager->stopAll();
    delete g_manager;
    
    return 0;
}
