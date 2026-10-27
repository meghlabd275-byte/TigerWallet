# Makefile for the TigerWallet privacy module.
#
# `make test` builds and runs the test suite. It previously printed
# "Build complete - no tests defined" while there were no tests, which made the
# target look like it had passed.

CXX ?= g++
CXXFLAGS ?= -std=c++17 -O3 -Wall -Wextra -Wconversion -Wsign-conversion -I./include
LDFLAGS ?= -lssl -lcrypto -pthread

SRC_DIR := src
INCLUDE_DIR := include
TEST_DIR := tests
BUILD_DIR := build

SOURCES := $(wildcard $(SRC_DIR)/*.cpp)
OBJECTS := $(SOURCES:$(SRC_DIR)/%.cpp=$(BUILD_DIR)/%.o)
TARGET := $(BUILD_DIR)/libprivacy.a

TEST_SOURCES := $(wildcard $(TEST_DIR)/*.cpp)
TEST_BINARIES := $(TEST_SOURCES:$(TEST_DIR)/%.cpp=$(BUILD_DIR)/%)

.PHONY: all clean test

all: $(TARGET)

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(TARGET): $(BUILD_DIR) $(OBJECTS)
	@echo "Building privacy library..."
	@ar rcs $@ $(OBJECTS)
	@echo "Library built: $@"

$(BUILD_DIR)/%.o: $(SRC_DIR)/%.cpp
	@mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) -c $< -o $@

# Each test file is a standalone program linked against the module sources.
$(BUILD_DIR)/%: $(TEST_DIR)/%.cpp $(SOURCES)
	@mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) $< $(SOURCES) -o $@ $(LDFLAGS)

test: $(TEST_BINARIES)
	@failed=0; \
	for t in $(TEST_BINARIES); do \
		echo "=== $$t ==="; \
		./$$t || failed=1; \
	done; \
	if [ $$failed -ne 0 ]; then echo "TESTS FAILED"; exit 1; fi; \
	echo "All privacy tests passed."

clean:
	rm -rf $(BUILD_DIR)
