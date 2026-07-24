#ifndef TIGER_MERKLE_H
#define TIGER_MERKLE_H
#include <string>
#include <vector>
namespace tiger {
std::string merkle_root(const std::vector<std::string>& data);
bool verify_proof(const std::string& root, const std::string& leaf, const std::vector<std::string>& proof);
}
#endif
