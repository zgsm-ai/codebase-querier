(use_declaration
  argument: (scoped_identifier
              name: (identifier) @use.name) *
  argument: (scoped_use_list
              list: (use_list) @use.names
              ) *
  ) @use

;; Rust structure query
;; Captures function definitions, struct definitions, trait definitions, and more

;; Function definitions
(function_item
  name: (identifier) @name) @function_item

;; Struct definitions
(struct_item
  name: (type_identifier) @name) @struct_item

;; Enum definitions
(enum_item
  name: (type_identifier) @name) @enum_item

;; Trait definitions
(trait_item
  name: (type_identifier) @name) @trait_item

;; Implementation blocks
(impl_item
  trait: (type_identifier) @name) @impl_item

;; Type definitions
(type_item
  name: (type_identifier) @name) @type_item

;; Constant definitions
(const_item
  name: (identifier) @name) @const_item

;; Static definitions
(static_item
  name: (identifier) @name) @static_item

;; Module declarations
(mod_item
  name: (identifier) @name) @class

;; Macro definitions
(macro_definition
  name: (identifier) @name) @definition.macro

(let_declaration
  pattern: (identifier) @local_variable.name
  ) @local_variable

(call_expression
  function: (identifier) @call.function.name
  arguments: (arguments) @call.function.arguments
  ) @call.function
