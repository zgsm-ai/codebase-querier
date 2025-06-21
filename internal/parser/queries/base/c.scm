(preproc_include
  "#include" @name
  ) @definition.include

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

(declaration
  declarator: (function_declarator
                declarator:  (identifier) @name)
  ) @declaration.function
