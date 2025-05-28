<?php

function greet($name) {
    echo "Hello, " . $name . "!\n";
}

class Greeter {
    public $greeting;

    public function __construct($message) {
        $this->greeting = $message;
    }

    public function greet() {
        return "Hello, " . $this->greeting;
    }
}

greet("PHP");

$greeter = new Greeter("world");
echo $greeter->greet() . "\n";

?> 