cmake_minimum_required(VERSION 3.15)
project(TigerWalletCore CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

# The previous source list named six files that did not exist, so this target
# could never link. It now lists the files that are actually implemented.
set(TIGER_CORE_SOURCES
  src/crypto.cpp
  src/transaction.cpp
  src/rpc.cpp
  src/mempool.cpp
  src/chain.cpp
  src/wallet.cpp
)

add_library(tigerwallet_core STATIC ${TIGER_CORE_SOURCES})
target_include_directories(tigerwallet_core PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)

find_package(OpenSSL REQUIRED)
find_package(CURL REQUIRED)
find_package(Threads REQUIRED)

target_link_libraries(tigerwallet_core
  PUBLIC
    OpenSSL::Crypto
    CURL::libcurl
    Threads::Threads
)

# libsecp256k1 is not shipped by every distribution's CMake config, so it is
# located directly. A missing library is a hard error: silently building without
# it would leave the signing entry points unresolved.
find_library(SECP256K1_LIBRARY NAMES secp256k1)
find_path(SECP256K1_INCLUDE_DIR NAMES secp256k1.h)
if(NOT SECP256K1_LIBRARY OR NOT SECP256K1_INCLUDE_DIR)
  message(FATAL_ERROR
    "libsecp256k1 was not found. Install it (Debian/Ubuntu: libsecp256k1-dev) "
    "before building tigerwallet_core.")
endif()

target_include_directories(tigerwallet_core PUBLIC ${SECP256K1_INCLUDE_DIR})
target_link_libraries(tigerwallet_core PUBLIC ${SECP256K1_LIBRARY})

# Warnings that catch real defects. These are errors because the encoding and
# signing paths are consensus-critical.
target_compile_options(tigerwallet_core PRIVATE
  -Wall -Wextra -Wpedantic
  -Wconversion -Wsign-conversion
  -Werror=return-type
  -Werror=uninitialized
)

install(TARGETS tigerwallet_core ARCHIVE DESTINATION lib)
install(DIRECTORY include/ DESTINATION include)

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------
option(TIGER_CORE_BUILD_TESTS "Build the tigerwallet_core test suite" ON)

if(TIGER_CORE_BUILD_TESTS)
  enable_testing()

  foreach(test_name crypto transaction wallet)
    add_executable(test_${test_name} tests/test_${test_name}.cpp)
    target_link_libraries(test_${test_name} PRIVATE tigerwallet_core)
    target_compile_options(test_${test_name} PRIVATE -Wall -Wextra)
    add_test(NAME ${test_name} COMMAND test_${test_name})
  endforeach()

  # A live check against a real node. It is not part of the default test run
  # because it needs network access and an endpoint; set TIGER_RPC_URL to run
  # it, e.g. ctest -R live_rpc --output-on-failure.
  add_executable(test_live_rpc tests/test_live_rpc.cpp)
  target_link_libraries(test_live_rpc PRIVATE tigerwallet_core)
  target_compile_options(test_live_rpc PRIVATE -Wall -Wextra)
  add_test(NAME live_rpc COMMAND test_live_rpc)
  set_tests_properties(live_rpc PROPERTIES DISABLED TRUE)
endif()
