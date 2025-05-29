#include <iostream>
#include <string>

void greet(const std::string& name) {
    std::cout << "Hello, " << name << "!" << std::endl;
}

class Greeter {
public:
    Greeter(const std::string& message) : greeting(message) {}

    std::string greet() const {
        return "Hello, " + greeting;
    }

private:
    std::string greeting;
};

int main() {
    greet("C++");

    Greeter greeter("world");
    std::cout << greeter.greet() << std::endl;

    return 0;
} 