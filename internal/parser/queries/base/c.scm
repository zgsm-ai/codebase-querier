(preproc_include
  "#include" @include.name
  ) @include

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

;; TODO 去找它的 identifier
;; variable & function  declaration
(declaration
  type: (_) @type
  ) @declaration

;; function_call  TODO ，这里不好确定它的parent，原因是存在嵌套、赋值等。可能得通过代码去递归。
(call_expression
  function: (identifier) @function.call.name
  arguments: (argument_list) @function.call.arguments
  ) @call.funciton

