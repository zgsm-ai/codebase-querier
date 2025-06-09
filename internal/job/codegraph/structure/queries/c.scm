(preproc_include ) @definition.include @name

(preproc_def )  @definition.macro @name

;; Constant declarations
(declaration
  (type_qualifier) @qualifier
  declarator: (init_declarator
                declarator: (identifier) @name)
  (#eq? @qualifier "const")) @definition.const


;; Variable declarations
(declaration
  declarator: (identifier) @name) @definition.variable

;; Function definitions
(function_definition
  declarator: (function_declarator
                declarator: (identifier) @name)) @definition.function

(declaration
  declarator: (function_declarator
                declarator:  (identifier) @name)
  ) @declaration.function


;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @struct_declaration

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @enum_declaration

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @union_declaration
