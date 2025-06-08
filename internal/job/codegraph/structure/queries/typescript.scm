;; TypeScript structure query
;; Captures function definitions, class definitions, interfaces, types, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function_declaration

;; Function expressions
(variable_declarator
  name: (identifier) @name
  value: (function_expression)) @variable_declarator  ;; 修正: function → function_expression

;; Arrow functions
(variable_declarator
  name: (identifier) @name
  value: (arrow_function)) @variable_declarator

;; Class declarations
(class_declaration
  name: (identifier) @name) @class_declaration

;; Class expressions
(variable_declarator
  name: (identifier) @name
  value: (class_expression)) @variable_declarator       ;; 修正: class → class_expression

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @method_definition

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface_declaration

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @type_alias_declaration

;; Type declarations（TypeScript 中通常用 type_alias_declaration 表示类型别名）
;; 注：type_declaration 可能不是标准节点，建议统一使用 type_alias_declaration

;; Enum declarations
(enum_declaration
  name: (type_identifier) @name) @enum_declaration

;; Namespace declarations
(namespace_declaration
  name: (identifier) @name) @namespace_declaration

;; Module declarations
(module_declaration
  name: (identifier) @name) @module_declaration

;; Variable declarations
(variable_declarator
  name: (identifier) @name) @variable_declarator

;; Constant declarations（建议通过修饰符判断，而非名称匹配）
(variable_declarator
  name: (identifier) @name
  parent: (variable_declaration
    declaration_specifiers: (modifier) @modifier
    (#eq? @modifier "const"))) @variable_declarator     ;; 修正: 基于 const 修饰符匹配

;; Generic type declarations
(type_parameter_declaration
  name: (type_identifier) @name) @type_parameter_declaration

;; Decorator declarations
(decorator
  expression: (call_expression
    function: (identifier) @name)) @decorator

;; Abstract class declarations（修正祖先节点判断逻辑）
(class_declaration
  name: (identifier) @name
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "abstract"))) @class_declaration     ;; 直接匹配 abstract 修饰符

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "abstract"))) @method_definition     ;; 直接匹配 abstract 修饰符

;; Mapped type declarations（可能需要调整节点路径）
(mapped_type
  type_identifier: (identifier) @name) @type_alias       ;; 假设 mapped_type 是正确节点

;; Conditional type declarations（通常嵌套在类型表达式中，需具体分析）
;; 注：conditional_type 可能不是顶层节点，需根据实际语法树调整

;; Import type declarations
(import_type
  name: (type_identifier) @name) @type_alias

;; Export type declarations
(export_type
  name: (type_identifier) @name) @type_alias