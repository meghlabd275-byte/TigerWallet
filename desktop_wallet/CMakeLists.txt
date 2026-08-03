cmake_minimum_required(VERSION 3.16)
project(TigerWalletDesktop VERSION 1.0.0 LANGUAGES CXX)

# Set C++ standard
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find required packages
find_package(CURL REQUIRED)
find_package(OpenSSL REQUIRED)

# Include directories
include_directories(
    ${CMAKE_SOURCE_DIR}/include
    ${CMAKE_SOURCE_DIR}/include/models
    ${CMAKE_SOURCE_DIR}/include/services
    ${OPENSSL_INCLUDE_DIR}
    ${CURL_INCLUDE_DIRS}
)

# Source files
set(MODEL_SOURCES
    src/models/wallet_models.cpp
)

set(SERVICE_SOURCES
    src/services/blockchain_service.cpp
    src/services/price_service.cpp
    src/services/swap_service.cpp
    src/services/staking_service.cpp
    src/services/nft_service.cpp
    src/services/api_client.cpp
    src/services/keychain_manager.cpp
    src/services/margin_trading_service.cpp
    src/services/futures_service.cpp
    src/services/copy_trading_service.cpp
    src/services/convert_service.cpp
    src/services/crypto_card_service.cpp
    src/services/p2p_trading_service.cpp
    src/services/fiat_ramp_service.cpp
)

set(ALL_SOURCES
    ${MODEL_SOURCES}
    ${SERVICE_SOURCES}
)

# Create library
add_library(tigerwallet_core STATIC ${ALL_SOURCES})

# Link libraries
target_link_libraries(tigerwallet_core
    PUBLIC OpenSSL::SSL
    PUBLIC OpenSSL::Crypto
    PUBLIC CURL::libcurl
)

# Add executable (for testing)
add_executable(tigerwallet_test
    src/main.cpp
)

target_link_libraries(tigerwallet_test
    PRIVATE tigerwallet_core
)

# Installation
install(TARGETS tigerwallet_core
    ARCHIVE DESTINATION lib
    LIBRARY DESTINATION lib
)

install(DIRECTORY include/
    DESTINATION include
)
