cmake_minimum_required(VERSION 3.16)
project(MasterWalletDesktop VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(CURL REQUIRED)
find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

include_directories(
    ${CMAKE_SOURCE_DIR}/src
    ${CMAKE_SOURCE_DIR}/src/services
    ${CMAKE_SOURCE_DIR}/src/ui
    ${OPENSSL_INCLUDE_DIR}
    ${CURL_INCLUDE_DIRS}
)

set(SERVICE_SOURCES
    src/services/api_client.cpp
    src/services/auth_service.cpp
    src/services/master_wallet_service.cpp
    src/services/web_socket_service.cpp
    src/services/account_abstraction_service.cpp
    src/services/paymaster_service.cpp
    src/services/privacy_service.cpp
    src/services/passkey_service.cpp
    src/services/super_admin_service.cpp
    src/services/tax_analytics_service.cpp
)

set(UI_SOURCES
    src/ui/theme.cpp
)

add_library(masterwallet_core STATIC ${SERVICE_SOURCES} ${UI_SOURCES})

target_link_libraries(masterwallet_core
    PUBLIC OpenSSL::SSL
    PUBLIC OpenSSL::Crypto
    PUBLIC CURL::libcurl
    PUBLIC Threads::Threads
)

add_executable(master_wallet_desktop
    src/main.cpp
)

target_link_libraries(master_wallet_desktop
    PRIVATE masterwallet_core
)
