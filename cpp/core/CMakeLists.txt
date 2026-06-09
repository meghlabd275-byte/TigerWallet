# TigerWallet C++ Core Library
# High-performance core operations for sub-millisecond latency

# Dependencies (use vcpkg or conan)
# Required: OpenSSL 3.0+, Boost 1.80+, LevelDB

# Build with:
# mkdir build && cd build
# cmake .. -DCMAKE_BUILD_TYPE=Release
# cmake --build . --config Release

cmake_minimum_required(VERSION 3.20)
project(TigerWalletCore VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# Optimization flags for maximum performance
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -O3 -march=native -mtune=native")
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -O3 -march=native -mtune=native")
    add_compile_options(-flto -fno-stack-protector -fomit-frame-pointer)
endif()

# Find dependencies
find_package(OpenSSL REQUIRED)
find_package(Boost REQUIRED COMPONENTS system thread))

# Include directories
include_directories(${CMAKE_SOURCE_DIR}/include)

# Source files
set(SOURCES
    src/wallet.cpp
    src/transaction.cpp
    src/crypto.cpp
    src/chain.cpp
    src/mempool.cpp
    src/rpc.cpp
)

set(HEADERS
    include/tigerwallet.hpp
    include/wallet.hpp
    include/transaction.hpp
    include/crypto.hpp
    include/chain.hpp
    include/mempool.hpp
    include/rpc.hpp
)

# Create shared library
add_library(tigerwallet_core SHARED ${SOURCES} ${HEADERS})

target_link_libraries(tigerwallet_core 
    PRIVATE 
        OpenSSL::Crypto 
        Boost::system 
        Boost::thread
        pthread
)

# Enable LTO
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    target_property(tigerwallet_core INTERFACE_LINK_OPTIONS 
        $<CONFIG:Release>:-flto)
endif()

# Installation
install(TARGETS tigerwallet_core DESTINATION lib)
install(FILES ${HEADERS} DESTINATION include/tigerwallet)
