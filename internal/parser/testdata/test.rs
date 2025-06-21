// 导入标准库模块
use std::fmt;
use std::fs::File;
use std::io::{self, Read};
use std::sync::mpsc;
use std::thread;

// 常量定义
const PI: f64 = 3.14159;

// 结构体
struct Rectangle {
    width: u32,
    height: u32,
}

// 枚举
enum Color {
    Red,
    Green,
    Blue,
}

// trait 定义
trait Area {
    fn area(&self) -> u32;
}

// 实现 trait
impl Area for Rectangle {
    fn area(&self) -> u32 {
        self.width * self.height
    }
}

// 泛型函数
fn largest<T: PartialOrd + Copy>(list: &[T]) -> T {
    let mut largest = list[0];
    for &item in list.iter() {
        if item > largest {
            largest = item;
        }
    }
    largest
}

// 闭包示例
fn closure_demo() {
    let list = vec![1, 2, 3];
    let only_borrows = || println!("From closure: {:?}", list);
    only_borrows();
}

// 错误处理
fn read_file() -> io::Result<String> {
    let mut file = File::open("hello.txt")?;
    let mut contents = String::new();
    file.read_to_string(&mut contents)?;
    Ok(contents)
}

// 生命周期注解
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() {
        x
    } else {
        y
    }
}

// 线程和通道
fn concurrency_demo() {
    let (tx, rx) = mpsc::channel();

    thread::spawn(move || {
        let val = String::from("hi");
        tx.send(val).unwrap();
    });

    let received = rx.recv().unwrap();
    println!("Got: {}", received);
}

// 主函数
fn main() {
    // 变量绑定
    let x = 5;
    let mut y = 10;
    y += x;

    // 控制流
    if x < y {
        println!("x is less than y");
    } else {
        println!("x is greater than or equal to y");
    }

    // 循环
    let mut counter = 0;
    let result = loop {
        counter += 1;
        if counter == 10 {
            break counter * 2;
        }
    };
    println!("Loop result: {}", result);

    // 模式匹配
    let color = Color::Red;
    match color {
        Color::Red => println!("The color is red"),
        Color::Green => println!("The color is green"),
        Color::Blue => println!("The color is blue"),
    }

    // 结构体实例
    let rect = Rectangle { width: 30, height: 50 };
    println!("Rectangle area: {}", rect.area());

    // 泛型使用
    let numbers = vec![34, 50, 25, 100, 65];
    println!("Largest number: {}", largest(&numbers));

    // 错误处理
    match read_file() {
        Ok(contents) => println!("File contents: {}", contents),
        Err(e) => println!("Couldn't read file: {}", e),
    }

    // 闭包调用
    closure_demo();

    // 生命周期使用
    let string1 = String::from("abcd");
    let string2 = "xyz";
    let result = longest(string1.as_str(), string2);
    println!("The longest string is {}", result);

    // 并发调用
    concurrency_demo();
}