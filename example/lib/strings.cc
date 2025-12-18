#include "lib/strings.h"
#include <algorithm>
#include <cctype>

namespace lib {

std::string reverse(const std::string& s) {
    std::string result = s;
    std::reverse(result.begin(), result.end());
    return result;
}

std::string to_upper(const std::string& s) {
    std::string result = s;
    std::transform(result.begin(), result.end(), result.begin(), ::toupper);
    return result;
}

std::string to_lower(const std::string& s) {
    std::string result = s;
    std::transform(result.begin(), result.end(), result.begin(), ::tolower);
    return result;
}

}  // namespace lib
