;; Java structure query
;; Captures class definitions, interface definitions, method definitions, and more

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @interface

;; Method declarations
(method_declaration
  name: (identifier) @name) @function

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @function

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @enum

;; Field declarations
(field_declaration
  declarator: (variable_declarator
    name: (identifier) @name)) @variable

;; Constant field declarations (static final)
(field_declaration
  (modifiers
    (modifier) @modifier1
    (modifier) @modifier2
    (#eq? @modifier1 "static")
    (#eq? @modifier2 "final"))
  declarator: (variable_declarator
    name: (identifier) @name)) @variable

;; Type parameter declarations (generics)
(type_parameter
  name: (identifier) @name) @type_alias 