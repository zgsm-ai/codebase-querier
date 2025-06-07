;; JavaScript structure query
;; Captures function definitions, class definitions, variable declarations, and more

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

;; Interface declarations (TypeScript)
(interface_declaration
  name: (type_identifier) @name) @interface

;; Type alias declarations (TypeScript)
(type_alias_declaration
  name: (type_identifier) @name) @type_alias

;; Variable declarations
(variable_declarator
  name: (identifier) @name) @variable_declarator

;; Constant declarations
(variable_declarator
  name: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable_declarator

;; Enum declarations (TypeScript)
(enum_declaration
  name: (type_identifier) @name) @enum 