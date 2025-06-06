;; JavaScript structure query
;; Captures function definitions, class definitions, variable declarations, and more

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

;; Interface declarations (TypeScript)
(interface_declaration
  name: (type_identifier) @name) @interface

;; Type alias declarations (TypeScript)
(type_alias_declaration
  name: (type_identifier) @name) @type_alias

;; Variable declarations
(variable_declarator
  name: (identifier) @name) @variable

;; Constant declarations
(variable_declarator
  name: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable

;; Enum declarations (TypeScript)
(enum_declaration
  name: (type_identifier) @name) @enum 