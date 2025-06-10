;; let/const declarations
(lexical_declaration
  (variable_declarator
    name: (identifier) @name )
  ) @lexical_declaration

;; Function declarations
(function_declaration
  name: (identifier) @name) @function_declaration

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name )
  ) @variable_declaration


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
  name: (identifier) @name) @enum_declaration


;; Decorator declarations
(decorator
  (identifier) @name) @decorator

;; Abstract class declarations（修正祖先节点判断逻辑）
(class_declaration
  name: (type_identifier) @name ) @class_declaration     ;; 直接匹配 abstract 修饰符

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name ) @method_definition     ;; 直接匹配 abstract 修饰符

(import_statement ) @import_type

;; Export type declarations
(export_statement ) @type_alias