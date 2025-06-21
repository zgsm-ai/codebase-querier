(preproc_include
  "#include" @name
  ) @definition.include

(preproc_def) @macro @name

;; Constant declarations
(declaration
  (type_qualifier) @qualifier
  declarator: (init_declarator
                declarator: (identifier) @name)
  (#eq? @qualifier "const")) @const


;; extern Variable declarations
(translation_unit
  (declaration
    (storage_class_specifier) @type
    (identifier) @name
    (#eq? @type "extern")
    ) @global_extern_variable
  )


;; Variable declarations
(translation_unit
  (declaration
    (_) * @type
    declarator: (init_declarator
                  declarator: (identifier) @name)
    (#not-eq? @type "const")
    (#not-eq? @type "extern")
    ) @global_variable
  )

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @declaration.struct

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @declaration.enum

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @declaration.union


(declaration
  declarator: (function_declarator
                declarator: (identifier) @name)
  ) @declaration.function

;; Function definitions
(function_definition
  declarator: (function_declarator
                declarator: (identifier) @name)) @definition.function


;; todo 去找它的 identifier , 基本数据类型，和其它类型不一样
(function_definition
  body: (_
          (declaration) @function.local_variable
          )
  )

;; function_call
(function_definition
  body: (_
          (expression_statement
            (call_expression
              function: (identifier) @function.call.name
              )
            )
          )
  )

