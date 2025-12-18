#include <iostream>
#include "lib/utils.h"
#include "lib/math.h"
#include "lib/strings.h"

int main() {
    std::cout << "Gangaji Example App" << std::endl;
    std::cout << "===================" << std::endl;

    // Math operations
    std::cout << "5 + 3 = " << lib::add(5, 3) << std::endl;
    std::cout << "5 * 3 = " << lib::multiply(5, 3) << std::endl;
    std::cout << "5! = " << lib::factorial(5) << std::endl;

    // String operations
    std::cout << "reverse('hello') = " << lib::reverse("hello") << std::endl;
    std::cout << "to_upper('hello') = " << lib::to_upper("hello") << std::endl;

    // Utils
    std::cout << "format_number(42) = " << lib::format_number(42) << std::endl;

    return 0;
}
