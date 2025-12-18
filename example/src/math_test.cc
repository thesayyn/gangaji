#include "lib/math.h"
#include <cassert>
#include <iostream>

int main() {
    // Test add
    assert(lib::add(2, 3) == 5);
    assert(lib::add(-1, 1) == 0);
    assert(lib::add(0, 0) == 0);

    // Test multiply
    assert(lib::multiply(2, 3) == 6);
    assert(lib::multiply(-2, 3) == -6);
    assert(lib::multiply(0, 100) == 0);

    // Test factorial
    assert(lib::factorial(0) == 1);
    assert(lib::factorial(1) == 1);
    assert(lib::factorial(5) == 120);
    assert(lib::factorial(10) == 3628800);

    std::cout << "All math tests passed!" << std::endl;
    return 0;
}
