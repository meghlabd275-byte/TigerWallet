# Makefile for TigerWallet Developer SDK

CXX = g++
CXXFLAGS = -std=c++17 -O3 -Wall -Wextra -I./include
LDFLAGS = -lcurl -lssl -lcrypto -pthread

SRC_DIR = src
BUILD_DIR = build

SOURCES = $(wildcard $(SRC_DIR)/*.cpp)
OBJECTS = $(SOURCES:$(SRC_DIR)/%.cpp=$(BUILD_DIR)/%.o)

all:
	mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) $(LDFLAGS) -c $(SRC_DIR)/developer_sdk.cpp -o $(BUILD_DIR)/developer_sdk.o

clean:
	rm -rf $(BUILD_DIR)
