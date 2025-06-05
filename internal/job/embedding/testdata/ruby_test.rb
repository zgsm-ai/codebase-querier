def greet(name)
  puts "Hello, #{name}!"
end

class Greeter
  def initialize(message)
    @greeting = message
  end

  def greet
    "Hello, #{@greeting}"
  end
end

greet("Ruby")

greeter = Greeter.new("world")
puts greeter.greet 