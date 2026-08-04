# Makefile for TigerWallet Privacy Features
# Ultra-low latency C++ implementation

CXX = g++
CXXFLAGS = -std=c++17 -O3 -Wall -Wextra -I./include
LDFLAGS = -lssl -lcrypto -pthread

SRC_DIR = src
INCLUDE_DIR = include
BUILD_DIR = build

SOURCES = $(wildcard $(SRC_DIR)/*.cpp)
OBJECTS = $(SOURCES:$(SRC_DIR)/%.cpp=$(BUILD_DIR)/%.o)
TARGET = $(BUILD_DIR)/libprivacy.a

.PHONY: all clean test

all: directories $(TARGET)

directories:
	@mkdir -p $(BUILD_DIR)

$(TARGET): $(OBJECTS)
	@echo "Building privacy library..."
	@ar rcs $@ $^
	@echo "Library built: $@"

$(BUILD_DIR)/%.o: $(SRC_DIR)/%.cpp $(INCLUDE_DIR)/*.hpp
	@echo "Compiling: $<"
	$(CXX) $(CXXFLAGS) -c $< -o $@

clean:
	rm -rf $(BUILD_DIR)

test:
	@echo "Build complete - no tests defined"
