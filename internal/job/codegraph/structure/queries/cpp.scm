;; C++ structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Function definitions
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @function

;; Class declarations
(class_specifier
  name: (type_identifier) @name) @class

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @struct

;; Method definitions (member functions)
(function_definition
  declarator: (function_declarator
    declarator: (qualified_identifier
      name: (identifier) @name))) @method

;; Constructor definitions
(function_definition
  declarator: (function_declarator
    declarator: (qualified_identifier
      name: (identifier) @name))
  (#match? @name "^[A-Z]")) @constructor

;; Namespace definitions
(namespace_definition
  name: (identifier) @name) @namespace

;; Template declarations
(template_declaration
  declaration: (declaration
    declarator: (identifier) @name)) @template

;; Using declarations
(using_declaration
  name: (identifier) @name) @using

;; Variable declarations
(declaration
  declarator: (init_declarator
    declarator: (identifier) @name)) @variable

;; Member variable declarations
(field_declaration
  declarator: (field_identifier) @name) @field

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @union

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @enum

;; Type alias declarations (using)
(alias_declaration
  name: (type_identifier) @name) @type_alias

;; Typedef declarations
(type_definition
  declarator: (type_identifier) @name) @typedef