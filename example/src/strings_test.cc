#include "lib/strings.h"
#include <cassert>
#include <iostream>

int main() {
    // Test reverse
    assert(lib::reverse("hello") == "olleh");
    assert(lib::reverse("") == "");
    assert(lib::reverse("a") == "a");

    // Test to_upper
    assert(lib::to_upper("hello") == "HELLO");
    assert(lib::to_upper("Hello World") == "HELLO WORLD");
    assert(lib::to_upper("") == "");

    // Test to_lower
    assert(lib::to_lower("HELLO") == "hello");
    assert(lib::to_lower("Hello World") == "hello world");
    assert(lib::to_lower("") == "");

    std::cout << "All strings tests passed!" << std::endl;
    return 0;
}
