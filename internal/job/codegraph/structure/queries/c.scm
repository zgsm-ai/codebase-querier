;; C structure query
;; Captures function definitions, struct definitions, variable declarations, and more

;; Function definitions
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @function_definition

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @struct_declaration

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @union_declaration

;; Variable declarations
(declaration
  declarator: (identifier) @name) @declaration

;; Constant declarations
(declaration
  (type_qualifier) @qualifier
  declarator: (init_declarator
    declarator: (identifier) @name)
  (#eq? @qualifier "const")) @variable

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @enum_declaration

;; Type definitions (typedef)
(typedef_declaration
  declarator: (type_identifier) @name) @typedef_declaration