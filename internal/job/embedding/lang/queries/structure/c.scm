;; C structure query
;; Captures function definitions, struct definitions, variable declarations, and more

;; Function definitions
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @function

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @struct

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @struct

;; Variable declarations
(declaration
  declarator: (init_declarator
    declarator: (identifier) @name)) @variable

;; Constant declarations
(declaration
  (type_qualifier) @qualifier
  declarator: (init_declarator
    declarator: (identifier) @name)
  (#eq? @qualifier "const")) @variable

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @enum

;; Type definitions (typedef)
(type_definition
  declarator: (type_identifier) @name) @type_alias

;; Function declarations (prototypes)
(declaration
  declarator: (function_declarator
    declarator: (identifier) @name)) @function 