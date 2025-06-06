;; C++ structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Class declarations
(class_specifier
  name: (type_identifier) @name) @class

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @struct

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @struct

;; Function declarations
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @function

;; Method declarations (inside classes)
(function_definition
  declarator: (function_declarator
    declarator: (field_identifier) @name)) @function

;; Constructor declarations
(function_definition
  declarator: (function_declarator
    declarator: (type_identifier) @name)) @function

;; Destructor declarations
(function_definition
  declarator: (function_declarator
    declarator: (destructor_name) @name)) @function

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

;; Type alias declarations (using)
(type_alias_declaration
  name: (type_identifier) @name) @type_alias

;; Typedef declarations
(type_definition
  declarator: (type_identifier) @name) @type_alias 