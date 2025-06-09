// 1. 变量声明与数据类型
const PI = 3.14159;
let number = 42;
let message = "Hello, JavaScript!";
let isDone = false;
let undef;
let nul = null;
let obj = { name: "John", age: 30 };
let arr = [1, 2, 3, 4, 5];

// 2. 函数与箭头函数
function add(a, b) {
    return a + b;
}

const multiply = (a, b) => a * b;

// 3. 类与继承
class Shape {
    constructor(name) {
        this.name = name;
    }

    area() {
        throw new Error("Method not implemented");
    }

    describe() {
        return `This is a ${this.name}`;
    }
}

class Circle extends Shape {
    constructor(radius) {
        super("Circle");
        this.radius = radius;
    }

    area() {
        return PI * this.radius ** 2;
    }
}

// 4. 解构赋值
const { name, age } = obj;
const [first, second, ...rest] = arr;

// 5. 模板字符串
console.log(`${message} The result is ${add(5, 3)}`);

// 6. 数组方法
const doubled = arr.map(n => n * 2);
const sum = arr.reduce((acc, n) => acc + n, 0);
const even = arr.filter(n => n % 2 === 0);

// 7. 对象方法与简写
const person = {
    name,
    age,
    greet() {
        return `Hi, I'm ${this.name}`;
    }
};

// 8. 异步操作 (Promise)
function fetchData() {
    return new Promise((resolve, reject) => {
        setTimeout(() => {
            resolve("Data loaded");
        }, 1000);
    });
}

// 9. 模块化 (ES6)
// 注意：在浏览器中需要通过 <script type="module"> 加载
export const utils = {
    square: n => n * n,
    cube: n => n ** 3
};

// 10. 事件处理 (模拟 DOM 事件)
class EventEmitter {
    constructor() {
        this.events = {};
    }

    on(event, callback) {
        if (!this.events[event]) {
            this.events[event] = [];
        }
        this.events[event].push(callback);
    }

    emit(event, data) {
        if (this.events[event]) {
            this.events[event].forEach(callback => callback(data));
        }
    }
}

// 11. 迭代器与生成器
function* counter() {
    let i = 0;
    while (true) {
        yield i++;
    }
}

// 12. 错误处理
try {
    throw new Error("Something went wrong");
} catch (error) {
    console.error(error.message);
} finally {
    console.log("This always runs");
}

// 13. 可选链与空值合并
const user = {
    profile: {
        email: "john@example.com"
    }
};

const email = user?.profile?.email ?? "No email provided";

// 14. 闭包
function outer() {
    const x = 10;
    return function inner() {
        return x;
    };
}

// 15. 应用示例
const circle = new Circle(5);
console.log(circle.describe());
console.log(`Area: ${circle.area()}`);

fetchData().then(data => {
    console.log(data);
}).catch(error => {
    console.error(error);
});

const emitter = new EventEmitter();
emitter.on("message", (msg) => console.log(`Received: ${msg}`));
emitter.emit("message", "Hello from event!");

const count = counter();
console.log(count.next().value); // 0
console.log(count.next().value); // 1