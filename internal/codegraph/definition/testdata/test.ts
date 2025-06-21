// 1. 导入模块
import { format } from "./utils"; // 假设存在utils模块提供format函数

// 2. 枚举类型
enum Status {
    Active = "ACTIVE",
    Inactive = "INACTIVE",
    Pending = "PENDING"
}

// 3. 基础接口
interface User {
    id: number;
    name: string;
    age: number;
    status: Status;
    createdAt: Date;
}

// 4. 映射类型 - 只读版本
type ReadonlyUser = {
    readonly [P in keyof User]: User[P];
};

// 5. 映射类型 - 可选版本
type PartialUser = {
    [P in keyof User]?: User[P];
};

// 6. 映射类型 - 选择特定属性
type UserInfo = {
    [P in "name" | "age"]: User[P];
};

// 7. 条件类型
type ExtractString<T> = {
    [P in keyof T]: T[P] extends string ? T[P] : never;
};

type UserStrings = ExtractString<User>;

// 8. 函数类型
function createUser(user: PartialUser): User {
    return {
        id: Math.random(),
        name: user.name || "Anonymous",
        age: user.age || 18,
        status: user.status || Status.Pending,
        createdAt: new Date()
    };
}

// 9. 泛型函数
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
    return obj[key];
}

// 10. 类
class UserManager {
    private users: User[] = [];

    addUser(user: User): void {
        this.users.push(user);
    }

    getUserById(id: number): User | undefined {
        return this.users.find(u => u.id === id);
    }

    updateUser(id: number, changes: PartialUser): void {
        const index = this.users.findIndex(u => u.id === id);
        if (index !== -1) {
            this.users[index] = { ...this.users[index], ...changes };
        }
    }

    getStatusCounts(): Record<Status, number> {
        return Object.values(Status).reduce((acc, status) => {
            acc[status] = this.users.filter(u => u.status === status).length;
            return acc;
        }, {} as Record<Status, number>);
    }
}

// 11. 类型守卫
function isActive(user: User): user is User & { status: Status.Active } {
    return user.status === Status.Active;
}

// 12. 联合类型
type UserWithRole = User & {
    role: "admin" | "editor" | "viewer";
};

// 13. 交叉类型
type AuditableUser = User & {
    createdBy: string;
    updatedAt: Date;
};

// 14. 字面量类型
type Permissions = "read" | "write" | "delete";

// 15. 可选链和空值合并
function formatUser(user: User | null | undefined): string {
    return format(`${user?.name ?? "Unknown"} (${user?.status ?? "N/A"})`);
}

// 16. 元组类型
type UserTuple = [id: number, name: string, age: number];

// 17. 命名空间
namespace UserUtils {
    export function isValidName(name: string): boolean {
        return name.length >= 2;
    }

    export function calculateAge(birthDate: Date): number {
        const today = new Date();
        return today.getFullYear() - birthDate.getFullYear();
    }
}

// 18. 装饰器
function logMethod(target: any, propertyKey: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    descriptor.value = function(...args: any[]) {
        console.log(`Calling ${propertyKey} with args: ${JSON.stringify(args)}`);
        const result = originalMethod.apply(this, args);
        console.log(`Method ${propertyKey} returned: ${result}`);
        return result;
    };
}

class Logger {
    @logMethod
    static info(message: string): void {
        console.log(`INFO: ${message}`);
    }
}

// 19. 索引签名
interface UserDictionary {
    [id: string]: User;
}

// 20. 应用示例
const manager = new UserManager();
const admin: UserWithRole = {
    ...createUser({ name: "Admin", status: Status.Active }),
    role: "admin"
};

manager.addUser(admin);
console.log(manager.getStatusCounts());
console.log(formatUser(manager.getUserById(admin.id)));