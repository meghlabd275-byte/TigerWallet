#ifndef GAS_SERVICE_HPP
#define GAS_SERVICE_HPP
#include <string>
using namespace std;
struct GasPrice { string chain; double slow, standard, fast; };
class GasService {
public:
    GasPrice getGasPrice(string c) { return {c, 10, 15, 25}; }
    double estimateGas(string c, string to, string data) { return 21000; }
};
#endif
