# 1. 模块导入
import math
from typing import List, Dict, Optional, Callable

# 2. 常量定义
PI = 3.14159

# 3. 函数定义
def add(a: int, b: int) -> int:
    """返回两个数的和"""
    return a + b

# 4. 类定义
class Shape:
    def __init__(self, name: str):
        self.name = name

    def area(self) -> float:
        raise NotImplementedError("Subclasses should implement this method")

    def describe(self) -> str:
        return f"This is a {self.name}"

# 5. 继承
class Circle(Shape):
    def __init__(self, radius: float):
        super().__init__("Circle")
        self.radius = radius

    def area(self) -> float:
        return PI * self.radius ** 2

# 6. 枚举类 (Python 3.4+)
from enum import Enum

class Color(Enum):
    RED = 1
    GREEN = 2
    BLUE = 3

# 7. 装饰器
def timer_decorator(func: Callable) -> Callable:
    import time
    def wrapper(*args, **kwargs):
        start = time.time()
        result = func(*args, **kwargs)
        end = time.time()
        print(f"{func.__name__} took {end - start} seconds")
        return result
    return wrapper

# 8. 生成器
def fibonacci_generator(n: int) -> Generator[int, None, None]:
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b

# 9. 数据结构操作
my_list: List[int] = [1, 2, 3, 4, 5]
my_dict: Dict[str, int] = {"a": 1, "b": 2, "c": 3}

# 10. 列表推导式
squared = [x**2 for x in my_list]

# 11. 条件表达式
message = "Even" if len(my_list) % 2 == 0 else "Odd"

# 12. 异常处理
try:
    result = 1 / 0
except ZeroDivisionError as e:
    print(f"Error: {e}")
finally:
    print("This always executes")

# 13. 上下文管理器
with open("example.txt", "w") as f:
    f.write("Hello, Python!")

# 14. 函数式编程
doubled = list(map(lambda x: x * 2, my_list))
evens = list(filter(lambda x: x % 2 == 0, my_list))

# 15. 内置高阶函数
sum_result = reduce(lambda a, b: a + b, my_list)  # 需要 from functools import reduce

# 16. 多线程
from threading import Thread

def print_numbers():
    for i in range(5):
        print(i)

thread = Thread(target=print_numbers)
thread.start()
thread.join()

# 17. 多进程
from multiprocessing import Process

def print_squares():
    for i in range(5):
        print(i**2)

process = Process(target=print_squares)
process.start()
process.join()

# 18. 异步编程 (Python 3.7+)
import asyncio

async def async_function():
    await asyncio.sleep(1)
    print("Async operation completed")

asyncio.run(async_function())

# 19. 类型注解
def greet(name: Optional[str] = None) -> str:
    return f"Hello, {name or 'World'}"

# 20. 类方法与静态方法
class Calculator:
    @staticmethod
    def add(a: int, b: int) -> int:
        return a + b

    @classmethod
    def multiply(cls, a: int, b: int) -> int:
        return a * b

# 21. 魔术方法
class Vector:
    def __init__(self, x: int, y: int):
        self.x = x
        self.y = y

    def __add__(self, other: 'Vector') -> 'Vector':
        return Vector(self.x + other.x, self.y + other.y)

    def __str__(self) -> str:
        return f"Vector({self.x}, {self.y})"

# 22. 模块与包结构
# 假设项目结构:
# mypackage/
#   __init__.py
#   utils.py
# 则可通过以下方式导入:
# from mypackage.utils import some_function

# 23. 测试框架 (简化示例)
def test_add():
    assert add(2, 3) == 5, "加法测试失败"

# 24. 元类 (简化示例)
class MyMeta(type):
    def __new__(cls, name, bases, attrs):
        attrs['custom_attribute'] = 'Added by metaclass'
        return super().__new__(cls, name, bases, attrs)

class MyClass(metaclass=MyMeta):
    pass

# 25. 装饰器类
class CountCalls:
    def __init__(self, func):
        self.func = func
        self.num_calls = 0

    def __call__(self, *args, **kwargs):
        self.num_calls += 1
        print(f"Call {self.num_calls} of {self.func.__name__}")
        return self.func(*args, **kwargs)

@CountCalls
def say_hello():
    print("Hello!")

# 26. 闭包
def outer_function(x: int) -> Callable[[int], int]:
    def inner_function(y: int) -> int:
        return x + y
    return inner_function

# 27. 命名元组
from collections import namedtuple

Point = namedtuple('Point', ['x', 'y'])
p = Point(10, 20)

# 28. 偏函数
from functools import partial

def power(base: int, exponent: int) -> int:
    return base ** exponent

square = partial(power, exponent=2)

# 29. 缓存装饰器
from functools import lru_cache

@lru_cache(maxsize=32)
def fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n-1) + fib(n-2)

# 30. 主程序入口
if __name__ == "__main__":
    circle = Circle(5.0)
    print(circle.describe())
    print(f"Area: {circle.area()}")

    for num in fibonacci_generator(5):
        print(num)

    print(squared)
    print(message)

    say_hello()
    say_hello()

    print(p.x, p.y)
    print(square(5))