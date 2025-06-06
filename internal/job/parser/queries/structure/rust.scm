;; Rust structure query
;; Captures struct definitions, function definitions, variable declarations, and more

;; Struct definitions
(struct_item
  name: (type_identifier) @name) @struct

;; Enum definitions
(enum_item
  name: (type_identifier) @name) @enum

;; Function definitions
(function_item
  name: (identifier) @name) @function

;; Method definitions (inside impl blocks)
(impl_item
  body: (declaration_list
    (function_item
      name: (identifier) @name))) @function

;; Trait definitions
(trait_item
  name: (type_identifier) @name) @interface

;; Type alias definitions
(type_item
  name: (type_identifier) @name) @type_alias

;; Constant declarations
(const_item
  name: (identifier) @name) @variable

;; Static declarations
(static_item
  name: (identifier) @name) @variable

;; Module declarations
(mod_item
  name: (identifier) @name) @class

;; Macro definitions
(macro_definition
  name: (identifier) @name) @function 