object HelloWorld {
  def main(args: Array[String]): Unit = {
    println("Hello, Scala!")
  }
}

class Greeter(greeting: String) {
  def greet(): String = {
    "Hello, " + greeting
  }
} 