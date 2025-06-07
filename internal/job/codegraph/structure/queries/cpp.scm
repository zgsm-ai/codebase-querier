;; C++ structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Function definitions
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @function_definition

;; Class declarations
(class_declaration
  name: (type_identifier) @name) @class_declaration

;; Struct declarations
(struct_declaration
  name: (type_identifier) @name) @struct_declaration

;; Method definitions
(function_definition
  declarator: (function_declarator
    declarator: (field_identifier) @name)) @function_definition

;; Constructor definitions
(function_definition
  declarator: (function_declarator
    declarator: (qualified_identifier
      name: (identifier) @name))) @function_definition

;; Namespace definitions
(namespace_definition
  name: (identifier) @name) @namespace_definition

;; Template declarations
(template_declaration
  declaration: (declaration
    declarator: (identifier) @name)) @template_declaration

;; Using declarations
(using_declaration
  name: (identifier) @name) @using_declaration

;; Variable declarations
(declaration
  declarator: (identifier) @name) @declaration

;; Member variable declarations
(field_declaration
  declarator: (field_identifier) @name) @field_declaration

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @struct

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @enum

;; Type alias declarations (using)
(type_alias_declaration
  name: (type_identifier) @name) @type_alias

;; Typedef declarations
(type_definition
  declarator: (type_identifier) @name) @type_alias 