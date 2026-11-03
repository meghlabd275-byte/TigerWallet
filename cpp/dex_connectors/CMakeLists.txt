cmake_minimum_required(VERSION 3.15)
project(TigerWalletDexConnectors CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

# The connectors call the node through tigerwallet_core's RpcClient and sign
# with its Transaction/Wallet helpers, so the core library is a hard dependency
# rather than a copy of that code.
if(NOT TARGET tigerwallet_core)
  add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/../core ${CMAKE_CURRENT_BINARY_DIR}/core)
endif()

add_library(tigerwallet_dex_connectors STATIC
  abi.cpp
  dex_connector.cpp
)

target_include_directories(tigerwallet_dex_connectors PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
target_link_libraries(tigerwallet_dex_connectors PUBLIC tigerwallet_core)

target_compile_options(tigerwallet_dex_connectors PRIVATE
  -Wall -Wextra -Wpedantic
  -Werror=return-type
  -Werror=uninitialized
)

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------
option(TIGER_DEX_BUILD_TESTS "Build the dex connector test suite" ON)

if(TIGER_DEX_BUILD_TESTS)
  enable_testing()
  add_executable(test_dex_connectors tests/test_dex_connectors.cpp)
  target_link_libraries(test_dex_connectors PRIVATE tigerwallet_dex_connectors)
  target_compile_options(test_dex_connectors PRIVATE -Wall -Wextra)
  add_test(NAME dex_connectors COMMAND test_dex_connectors)

  # A live check against a real node. Disabled by default because it needs
  # network access and an endpoint. Run with:
  #   TIGER_RPC_URL=https://... ctest -R dex_live --output-on-failure
  add_executable(test_dex_live tests/test_dex_live.cpp)
  target_link_libraries(test_dex_live PRIVATE tigerwallet_dex_connectors)
  target_compile_options(test_dex_live PRIVATE -Wall -Wextra)
  add_test(NAME dex_live COMMAND test_dex_live)
  set_tests_properties(dex_live PROPERTIES DISABLED TRUE)
endif()
