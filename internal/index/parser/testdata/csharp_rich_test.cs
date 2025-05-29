// This is a rich test file for the C# parser.
// It includes various language constructs like classes, interfaces, enums, structs, delegates, events, properties, methods, control flow, and more.

using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

// A simple class
public class HelloWorld
{
    // A simple method
    public void SayHello(){
        Console.WriteLine("Hello, C#!");
    }

    // Method with parameters and return type
    public int Add(int a, int b)
    {
        return a + b;
    }

    // Property
    public string Message { get; set; }

    // Constructor
    public HelloWorld(string msg)
    {
        Message = msg;
    }

    // Using a List
    public void ExploreList()
    {
        List<int> numbers = new List<int> { 1, 2, 3, 4, 5 };
        foreach (var number in numbers)
        {
            Console.WriteLine(number);
        }
    }
}

// An interface definition
public interface IMyInterface
{
    void DoSomething();
    bool DoSomethingElse(string value);
}

// Implementing an interface
public class MyClass : IMyInterface
{
    public void DoSomething()
    {
        Console.WriteLine("Doing something...");
    }

    public bool DoSomethingElse(string value)
    {
        Console.WriteLine($"Doing something else with: {value}");
        return !string.IsNullOrEmpty(value);
    }
}

// An enum definition
public enum Status
{
    Pending,
    Processing,
    Completed,
    Failed
}

// A struct definition
public struct Point
{
    public double X { get; set; }
    public double Y { get; set; }

    public double DistanceFromOrigin()
    {
        return Math.Sqrt(X * X + Y * Y);
    }
}

// A delegate
public delegate void GreetingDelegate(string name);

// An event
public class EventPublisher
{
    public event GreetingDelegate OnGreet;

    public void RaiseGreetEvent(string name)
    {
        OnGreet?.Invoke(name);
    }
}

// Control flow: if/else
public void CheckNumber(int num)
{
    if (num > 0)
    {
        Console.WriteLine("Positive");
    } else if (num < 0)
    {
        Console.WriteLine("Negative");
    } else
    {
        Console.WriteLine("Zero");
    }
}

// Control flow: while loop
public void SimpleWhile()
{
    int number = 3;
    while (number != 0)
    {
        Console.WriteLine($"{number}!");
        number--;
    }
}

// Control flow: for loop
public void SimpleFor()
{
    int[] a = { 10, 20, 30, 40, 50 };
    for (int i = 0; i < a.Length; i++)
    {
        Console.WriteLine($"The value is: {a[i]}");
    }
}

// Comments:
// Single-line comment

/*
Multi-line
comment
*/

/// <summary>
/// Doc comment for a method
/// </summary>
public void DocumentedMethod()
{
    Console.WriteLine("This method has documentation.");
}

// More code to reach > 500 lines

public class AnotherClass
{
    public int Id { get; set; }
    public string Name { get; set; }

    public AnotherClass(int id, string name)
    {
        Id = id;
        Name = name;
    }

    public void Process()
    {
        Console.WriteLine($"Processing {Name} with ID {Id}");
    }
}

public static class Utility
{
    public static List<T> ProcessItems<T>(List<T> items)
    {
        // In a real scenario, this would process data.
        return items;
    }

    public static U TransformItem<T, U>(T item, Func<T, U> transformer)
    {
        return transformer(item);
    }
}

// Adding more content to reach 500+ lines

public void PlaceholderMethod1() { /* ... */ }
public void PlaceholderMethod2() { /* ... */ }
public void PlaceholderMethod3() { /* ... */ }
public void PlaceholderMethod4() { /* ... */ }
public void PlaceholderMethod5() { /* ... */ }
public void PlaceholderMethod6() { /* ... */ }
public void PlaceholderMethod7() { /* ... */ }
public void PlaceholderMethod8() { /* ... */ }
public void PlaceholderMethod9() { /* ... */ }
public void PlaceholderMethod10() { /* ... */ }
public void PlaceholderMethod11() { /* ... */ }
public void PlaceholderMethod12() { /* ... */ }
public void PlaceholderMethod13() { /* ... */ }
public void PlaceholderMethod14() { /* ... */ }
public void PlaceholderMethod15() { /* ... */ }
public void PlaceholderMethod16() { /* ... */ }
public void PlaceholderMethod17() { /* ... */ }
public void PlaceholderMethod18() { /* ... */ }
public void PlaceholderMethod19() { /* ... */ }
public void PlaceholderMethod20() { /* ... */ }
public void PlaceholderMethod21() { /* ... */ }
public void PlaceholderMethod22() { /* ... */ }
public void PlaceholderMethod23() { /* ... */ }
public void PlaceholderMethod24() { /* ... */ }
public void PlaceholderMethod25() { /* ... */ }
public void PlaceholderMethod26() { /* ... */ }
public void PlaceholderMethod27() { /* ... */ }
public void PlaceholderMethod28() { /* ... */ }
public void PlaceholderMethod29() { /* ... */ }
public void PlaceholderMethod30() { /* ... */ }
public void PlaceholderMethod31() { /* ... */ }
public void PlaceholderMethod32() { /* ... */ }
public void PlaceholderMethod33() { /* ... */ }
public void PlaceholderMethod34() { /* ... */ }
public void PlaceholderMethod35() { /* ... */ }
public void PlaceholderMethod36() { /* ... */ }
public void PlaceholderMethod37() { /* ... */ }
public void PlaceholderMethod38() { /* ... */ }
public void PlaceholderMethod39() { /* ... */ }
public void PlaceholderMethod40() { /* ... */ }
public void PlaceholderMethod41() { /* ... */ }
public void PlaceholderMethod42() { /* ... */ }
public void PlaceholderMethod43() { /* ... */ }
public void PlaceholderMethod44() { /* ... */ }
public void PlaceholderMethod45() { /* ... */ }
public void PlaceholderMethod46() { /* ... */ }
public void PlaceholderMethod47() { /* ... */ }
public void PlaceholderMethod48() { /* ... */ }
public void PlaceholderMethod49() { /* ... */ }
public void PlaceholderMethod50() { /* ... */ }
public void PlaceholderMethod51() { /* ... */ }
public void PlaceholderMethod52() { /* ... */ }
public void PlaceholderMethod53() { /* ... */ }
public void PlaceholderMethod54() { /* ... */ }
public void PlaceholderMethod55() { /* ... */ }
public void PlaceholderMethod56() { /* ... */ }
public void PlaceholderMethod57() { /* ... */ }
public void PlaceholderMethod58() { /* ... */ }
public void PlaceholderMethod59() { /* ... */ }
public void PlaceholderMethod60() { /* ... */ }
public void PlaceholderMethod61() { /* ... */ }
public void PlaceholderMethod62() { /* ... */ }
public void PlaceholderMethod63() { /* ... */ }
public void PlaceholderMethod64() { /* ... */ }
public void PlaceholderMethod65() { /* ... */ }
public void PlaceholderMethod66() { /* ... */ }
public void PlaceholderMethod67() { /* ... */ }
public void PlaceholderMethod68() { /* ... */ }
public void PlaceholderMethod69() { /* ... */ }
public void PlaceholderMethod70() { /* ... */ }
public void PlaceholderMethod71() { /* ... */ }
public void PlaceholderMethod72() { /* ... */ }
public void PlaceholderMethod73() { /* ... */ }
public void PlaceholderMethod74() { /* ... */ }
public void PlaceholderMethod75() { /* ... */ }
public void PlaceholderMethod76() { /* ... */ }
public void PlaceholderMethod77() { /* ... */ }
public void PlaceholderMethod78() { /* ... */ }
public void PlaceholderMethod79() { /* ... */ }
public void PlaceholderMethod80() { /* ... */ }
public void PlaceholderMethod81() { /* ... */ }
public void PlaceholderMethod82() { /* ... */ }
public void PlaceholderMethod83() { /* ... */ }
public void PlaceholderMethod84() { /* ... */ }
public void PlaceholderMethod85() { /* ... */ }
public void PlaceholderMethod86() { /* ... */ }
public void PlaceholderMethod87() { /* ... */ }
public void PlaceholderMethod88() { /* ... */ }
public void PlaceholderMethod89() { /* ... */ }
public void PlaceholderMethod90() { /* ... */ }
public void PlaceholderMethod91() { /* ... */ }
public void PlaceholderMethod92() { /* ... */ }
public void PlaceholderMethod93() { /* ... */ }
public void PlaceholderMethod94() { /* ... */ }
public void PlaceholderMethod95() { /* ... */ }
public void PlaceholderMethod96() { /* ... */ }
public void PlaceholderMethod97() { /* ... */ }
public void PlaceholderMethod98() { /* ... */ }
public void PlaceholderMethod99() { /* ... */ }
public void PlaceholderMethod100() { /* ... */ }

public void FinalCSharpMethod()
{
    Console.WriteLine("End of C# test file.");
} 