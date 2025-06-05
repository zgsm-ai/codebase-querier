fun main() {
  val greeting = "Hello, Kotlin!"
  println(greeting)

  class Example {
    fun greet(name: String) {
      println("Hello, $name!")
    }
  }

  val example = Example()
  example.greet("World")
} 