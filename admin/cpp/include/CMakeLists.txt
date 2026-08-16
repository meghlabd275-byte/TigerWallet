cmake_minimum_required(VERSION 3.20)
project(TigerAdminCppCore VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Compiler optimizations for ultra-low latency
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -O3 -march=native -mtune=native")
    set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -ffast-math -funroll-loops")
    set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -flto -fno-exceptions")
endif()

# Find required packages
find_package(Threads REQUIRED)
find_package(OpenSSL REQUIRED)
find_package(Protobuf REQUIRED)

# Include directories
include_directories(
    ${CMAKE_SOURCE_DIR}/include
    ${OPENSSL_INCLUDE_DIR}
    ${PROTOBUF_INCLUDE_DIR}
)

# Source files
set(SOURCES
    src/admin_core.cpp
    src/admin_server.cpp
    src/admin_handler.cpp
    src/admin_auth.cpp
    src/admin_kyc.cpp
    src/admin_transactions.cpp
    src/admin_tokens.cpp
    src/admin_blockchain.cpp
    src/admin_analytics.cpp
    src/admin_websocket.cpp
    src/admin_cache.cpp
    src/admin_database.cpp
    src/admin_security.cpp
    src/admin_audit.cpp
    src/admin_rate_limiter.cpp
)

# Headers
set(HEADERS
    include/admin_core.hpp
    include/admin_server.hpp
    include/admin_handler.hpp
    include/admin_auth.hpp
    include/admin_kyc.hpp
    include/admin_transactions.hpp
    include/admin_tokens.hpp
    include/admin_blockchain.hpp
    include/admin_analytics.hpp
    include/admin_websocket.hpp
    include/admin_cache.hpp
    include/admin_database.hpp
    include/admin_security.hpp
    include/admin_audit.hpp
    include/admin_rate_limiter.hpp
    include/admin_models.hpp
    include/admin_config.hpp
    include/admin_logger.hpp
    include/admin_connection_pool.hpp
    include/admin_domain.hpp
    include/admin_domains.hpp
    include/admin_futures.cpp
    include/admin_options.cpp
    include/admin_copy_trading.cpp
    include/admin_convert.cpp
    include/admin_onramp.cpp
    include/admin_offramp.cpp
    include/admin_p2p_clients.cpp
    include/admin_partners.cpp
    include/admin_rewards.cpp
    include/admin_marketing.cpp
    include/admin_roles.cpp
    include/admin_p2p_merchants.cpp
)

# Create executable
add_executable(tiger_admin_cpp
    ${SOURCES}
    ${HEADERS}
)

# Link libraries
target_link_libraries(tiger_admin_cpp
    Threads::Threads
    OpenSSL::SSL
    OpenSSL::Crypto
    protobuf::libprotobuf
    pthread
    dl
)

# Installation
install(TARGETS tiger_admin_cpp DESTINATION bin)
install(DIRECTORY include/ DESTINATION include/tiger_admin)
