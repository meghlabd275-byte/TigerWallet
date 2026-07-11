cmake_minimum_required(VERSION 3.20)
project(TigerWalletBundler VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# Find required packages
find_package(Threads REQUIRED)
find_package(OpenSSL REQUIRED)
find_package(Boost REQUIRED COMPONENTS system thread filesystem)

# Include directories
include_directories(
    ${CMAKE_SOURCE_DIR}/include
    ${Boost_INCLUDE_DIRS}
    ${OPENSSL_INCLUDE_DIR}
)

# Source files
set(SOURCES
    src/main.cpp
    src/bundler.cpp
    src/userop_validation.cpp
    src/mempool.cpp
    src/gas_estimator.cpp
    src/reputation.cpp
)

# Header files
set(HEADERS
    include/bundler.hpp
    include/userop.hpp
    include/mempool.hpp
    include/gas_estimator.hpp
    include/reputation.hpp
    include/signature_validator.hpp
)

# Create executable
add_executable(tiger_bundler ${SOURCES} ${HEADERS})

# Link libraries
target_link_libraries(tiger_bundler
    Threads::Threads
    OpenSSL::Crypto
    Boost::system
    Boost::thread
    Boost::filesystem
)

# Installation
install(TARGETS tiger_bundler DESTINATION bin)
install(FILES ${HEADERS} DESTINATION include/tiger_bundler)

# Testing
enable_testing()
add_subdirectory(tests)
