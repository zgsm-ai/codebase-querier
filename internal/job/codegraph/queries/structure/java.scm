;; Java structure query
;; Captures class definitions, interface definitions, method definitions, and more

;; Class declarations
(class_declaration
  name: (identifier) @name) @class_declaration

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @interface_declaration

;; Method declarations
(method_declaration
  name: (identifier) @name) @method_declaration

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @constructor_declaration

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @enum_declaration

;; Field declarations
(field_declaration
  declarator: (variable_declarator
    name: (identifier) @name)) @field_declaration

;; Constant field declarations (static final)
(field_declaration
  (modifiers
    (modifier) @modifier1
    (modifier) @modifier2
    (#eq? @modifier1 "static")
    (#eq? @modifier2 "final"))
  declarator: (variable_declarator
    name: (identifier) @name)) @field_declaration

;; Type parameter declarations (generics)
(type_parameter
  name: (identifier) @name) @type_parameter 