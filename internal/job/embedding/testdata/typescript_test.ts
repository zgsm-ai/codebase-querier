function greet(name: string): void {
  console.log(`Hello, ${name}!`);
}

class Greeter {
  greeting: string;

  constructor(message: string) {
    this.greeting = message;
  }

  greet(): string {
    return "Hello, " + this.greeting;
  }
}

greet("TypeScript");

let greeter = new Greeter("world"); 