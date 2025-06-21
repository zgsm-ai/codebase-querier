// 1. 模块与导入
mod math {
    pub const PI: f64 = 3.14159;

    pub fn square(x: f64) -> f64 {
        x * x
    }
}

// 2. 结构体
struct Point {
    x: f64,
    y: f64,
}

// 3. 枚举
enum Color {
    Red,
    Green,
    Blue,
}

// 4. 泛型结构体
struct Container<T> {
    item: T,
}

// 5. trait 定义
trait Shape {
    fn area(&self) -> f64;
    fn name(&self) -> &'static str;
}

// 6. 实现 trait
struct Circle {
    radius: f64,
}

impl Shape for Circle {
    fn area(&self) -> f64 {
        math::PI * math::square(self.radius)
    }

    fn name(&self) -> &'static str {
        "Circle"
    }
}

// 7. 方法实现
impl Point {
    fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    fn distance_from_origin(&self) -> f64 {
        (self.x.powi(2) + self.y.powi(2)).sqrt()
    }
}

// 8. 生命周期注解
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() {
        x
    } else {
        y
    }
}

// 9. 错误处理
fn divide(a: f64, b: f64) -> Result<f64, &'static str> {
    if b == 0.0 {
        Err("Division by zero")
    } else {
        Ok(a / b)
    }
}

// 10. 闭包
fn apply<F>(f: F, x: f64) -> f64
where
    F: Fn(f64) -> f64,
{
    f(x)
}

// 11. 迭代器
fn sum_squares(n: u32) -> u32 {
    (1..=n).map(|x| x * x).sum()
}

// 12. 模式匹配
fn get_color_name(color: Color) -> &'static str {
    match color {
        Color::Red => "Red",
        Color::Green => "Green",
        Color::Blue => "Blue",
    }
}

// 13. 所有权与借用
fn take_ownership(s: String) {
    println!("Got: {}", s);
}

fn borrow_slice(s: &str) {
    println!("Borrowed: {}", s);
}

// 14. 智能指针
use std::rc::Rc;

fn count_rc() {
    let a = Rc::new(5);
    let b = Rc::clone(&a);
    let c = Rc::clone(&a);

    println!("Reference count: {}", Rc::strong_count(&a));
}

// 15. 并发
use std::thread;

fn run_threads() {
    let handles: Vec<_> = (0..5).map(|i| {
        thread::spawn(move || {
            println!("Thread {} says hello!", i);
        })
    }).collect();

    for handle in handles {
        handle.join().unwrap();
    }
}

// 16. 泛型函数
fn largest<T: PartialOrd>(list: &[T]) -> &T {
    let mut largest = &list[0];
    for item in list {
        if item > largest {
            largest = item;
        }
    }
    largest
}

// 17. 模块可见性
mod private {
    fn private_function() {
        println!("This is private");
    }

    pub fn public_wrapper() {
        private_function();
    }
}

// 18. 结构体方法链
struct Counter {
    value: u32,
}

impl Counter {
    fn new() -> Self {
        Counter { value: 0 }
    }

    fn increment(&mut self) -> &mut Self {
        self.value += 1;
        self
    }

    fn decrement(&mut self) -> &mut Self {
        self.value -= 1;
        self
    }

    fn get(&self) -> u32 {
        self.value
    }
}

// 19. 关联类型
trait Iterator {
    type Item;

    fn next(&mut self) -> Option<Self::Item>;
}

// 新增: 20. 宏定义 - 计算阶乘
macro_rules! factorial {
    // 基础情况: 0 的阶乘是 1
    (0) => { 1 };
    // 递归情况: n! = n * (n-1)!
    ($n:expr) => { $n * factorial!($n - 1) };
}

// 21. 主函数
fn main() {
    // 测试 Point
    let p = Point::new(3.0, 4.0);
    println!("Distance from origin: {}", p.distance_from_origin());

    // 测试 Circle
    let circle = Circle { radius: 5.0 };
    println!("{} area: {}", circle.name(), circle.area());

    // 测试闭包
    let result = apply(|x| x * x, 10.0);
    println!("Square: {}", result);

    // 测试迭代器
    println!("Sum of squares: {}", sum_squares(5));

    // 测试模式匹配
    println!("Color: {}", get_color_name(Color::Blue));

    // 测试错误处理
    match divide(10.0, 2.0) {
        Ok(result) => println!("Result: {}", result),
        Err(e) => println!("Error: {}", e),
    }

    // 测试所有权
    let s = String::from("hello");
    borrow_slice(&s);
    take_ownership(s); // s 被移动

    // 测试并发
    run_threads();

    // 测试方法链
    let count = Counter::new().increment().increment().decrement().get();
    println!("Count: {}", count);

    // 测试智能指针
    count_rc();

    // 测试宏
    println!("Factorial of 5: {}", factorial!(5));
    println!("Factorial of 10: {}", factorial!(10));
}