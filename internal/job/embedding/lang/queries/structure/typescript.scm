;; TypeScript structure query
;; Captures function definitions, class definitions, interfaces, types, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function

;; Function expressions
(variable_declarator
  name: (identifier) @name
  value: (function)) @function

;; Arrow functions
(variable_declarator
  name: (identifier) @name
  value: (arrow_function)) @function

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Class expressions
(variable_declarator
  name: (identifier) @name
  value: (class)) @class

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @function

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @type_alias

;; Type declarations
(type_declaration
  name: (type_identifier) @name) @type_alias

;; Enum declarations
(enum_declaration
  name: (type_identifier) @name) @enum

;; Namespace declarations
(namespace_declaration
  name: (identifier) @name) @class

;; Module declarations
(module_declaration
  name: (identifier) @name) @class

;; Variable declarations
(variable_declarator
  name: (identifier) @name) @variable

;; Constant declarations
(variable_declarator
  name: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable

;; Generic type declarations
(type_parameter_declaration
  name: (type_identifier) @name) @type_alias

;; Decorator declarations
(decorator
  expression: (call_expression
    function: (identifier) @name)) @function

;; Abstract class declarations
(class_declaration
  name: (identifier) @name
  (#has-ancestor? @name abstract_class_declaration)) @class

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  (#has-ancestor? @name abstract_method_declaration)) @function

;; Mapped type declarations
(mapped_type_clause
  name: (type_identifier) @name) @type_alias

;; Conditional type declarations
(conditional_type
  name: (type_identifier) @name) @type_alias

;; Import type declarations
(import_type
  name: (type_identifier) @name) @type_alias

;; Export type declarations
(export_type
  name: (type_identifier) @name) @type_alias 