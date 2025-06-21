(preproc_include
  "#include" @include.name
  ) @include


;; Class declarations
(class_specifier
  name: (type_identifier) @name) @definition.class

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @definition.struct


;; Variable declarations (keep as declaration)
(declaration
  declarator: (init_declarator
                declarator: (identifier) @name)) @variable

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

(function_definition
  declarator: (function_declarator
                declarator: (identifier) @name)) @definition.function

;; TODO 对象.方法
(call_expression
  function: (
              field_expression
              argument: (identifier) @method.call.object
              field: (field_identifier) @method.call.name
              )
  arguments: (argument_list) @method.call.arguments
  ) @call.method

;; 函数调用
(call_expression
  function: (qualified_identifier
              scope: (namespace_identifier) @call.function.namespace
              name: (identifier) @call.function.name
              )
  arguments: (argument_list) @call.function.arguments
  ) @call.function