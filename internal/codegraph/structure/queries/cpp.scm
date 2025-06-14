;; C++ structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Function definitions
(function_definition
  declarator: (function_declarator
                declarator: (identifier) @name)) @definition.function

;; Class declarations
(class_specifier
  name: (type_identifier) @name) @definition.class

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @definition.struct

;; Method definitions (member functions)
(function_definition
  declarator: (function_declarator
                declarator: (qualified_identifier
                               name: (identifier) @name))) @definition.method

;; Constructor definitions
(function_definition
  declarator: (function_declarator
                declarator: (qualified_identifier
                              name: (identifier) @name))
  (#match? @name "^[A-Z]")) @definition.constructor


;; Template declarations (these are definitions)
(template_declaration
  (function_definition
    declarator: (identifier) @name)) @declaration.template

;; Variable declarations (keep as declaration)
(declaration
  declarator: (init_declarator
                declarator: (identifier) @name)) @declaration.variable

;; Member variable declarations (keep as declaration)
(field_declaration
  declarator: (field_identifier) @name) @declaration.field

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @definition.union

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @definition.enum

;; Type alias declarations (these are definitions)
(alias_declaration
  name: (type_identifier) @name) @definition.type_alias

;; Typedef declarations
(type_definition
  declarator: (type_identifier) @name) @definition.typedef