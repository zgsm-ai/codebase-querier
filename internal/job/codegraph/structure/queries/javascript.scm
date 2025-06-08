;; JavaScript structure query
;; Captures function definitions, class definitions, variable declarations, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (function_expression))) @function

;; Arrow functions
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (arrow_function))) @function

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Class expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (class))) @class

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @method

;; Variable declarations
(variable_declaration
  (variable_declarator
    name: (identifier) @name)) @variable

;; Constant declarations (const)
(lexical_declaration
  "const"
  (variable_declarator
    name: (identifier) @name)) @constant

;; Object properties
(pair
  key: (property_identifier) @name) @property

;; Export declarations
(export_statement
  declaration: (function_declaration
    name: (identifier) @name)) @export

;; Export named declarations
(export_statement
  (export_clause
    (export_specifier
      name: (identifier) @name))) @export