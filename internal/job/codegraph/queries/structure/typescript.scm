;; TypeScript structure query
;; Captures function definitions, class definitions, interfaces, types, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function_declaration

;; Function expressions
(variable_declarator
  name: (identifier) @name
  value: (function)) @variable_declarator

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
  value: (class)) @variable_declarator

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @method_definition

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface_declaration

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @type_alias_declaration

;; Type declarations
(type_declaration
  name: (type_identifier) @name) @type_declaration

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

;; Constant declarations
(variable_declarator
  name: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable_declarator

;; Generic type declarations
(type_parameter_declaration
  name: (type_identifier) @name) @type_parameter_declaration

;; Decorator declarations
(decorator
  expression: (call_expression
    function: (identifier) @name)) @decorator

;; Abstract class declarations
(class_declaration
  name: (identifier) @name
  (#has-ancestor? @name abstract_class_declaration)) @class_declaration

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  (#has-ancestor? @name abstract_method_declaration)) @method_definition

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