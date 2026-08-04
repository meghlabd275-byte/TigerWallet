# Makefile for TigerWallet White Label SDK
# Ultra-low latency C++ implementation

CXX = g++
CXXFLAGS = -std=c++17 -O3 -Wall -Wextra -I./include
LDFLAGS = -lcurl -lssl -lcrypto -ljsoncpp -pthread

# Directories
SRC_DIR = src
INCLUDE_DIR = include
BUILD_DIR = build
BIN_DIR = bin

# Source files
SOURCES = $(wildcard $(SRC_DIR)/*.cpp)
OBJECTS = $(SOURCES:$(SRC_DIR)/%.cpp=$(BUILD_DIR)/%.o)
TARGET = $(BIN_DIR)/libwhite_label_sdk.a

# Dependencies
DEPS = $(wildcard $(INCLUDE_DIR)/*.hpp)

.PHONY: all clean test install uninstall

all: directories $(TARGET)

directories:
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(BIN_DIR)

$(TARGET): $(OBJECTS)
	@echo "Building static library..."
	@ar rcs $@ $^
	@echo "Library built successfully: $@"

$(BUILD_DIR)/%.o: $(SRC_DIR)/%.cpp $(DEPS)
	@echo "Compiling: $<"
	$(CXX) $(CXXFLAGS) -c $< -o $@

clean:
	@echo "Cleaning build files..."
	rm -rf $(BUILD_DIR) $(BIN_DIR)
	@echo "Clean complete"

test: $(TARGET)
	@echo "Running tests..."
	@echo "Build complete - no tests defined yet"

install: $(TARGET)
	@echo "Installing SDK..."
	cp -r $(INCLUDE_DIR) /usr/local/include/tigerwallet_wl_sdk
	cp $(TARGET) /usr/local/lib/
	@echo "Install complete"

uninstall:
	@echo "Uninstalling SDK..."
	rm -rf /usr/local/include/tigerwallet_wl_sdk
	rm -f /usr/local/lib/libwhite_label_sdk.a
	@echo "Uninstall complete"

.PHONY: all clean test install uninstall
