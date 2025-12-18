#include "lib/utils.h"
#include "lib/math.h"
#include "lib/strings.h"
#include <sstream>

namespace lib {

std::string format_number(int n) {
    std::ostringstream oss;
    oss << n;
    return oss.str();
}

int parse_number(const std::string& s) {
    std::istringstream iss(s);
    int n;
    iss >> n;
    return n;
}

}  // namespace lib
