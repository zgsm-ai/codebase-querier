;; Kotlin structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Class definitions
(class_declaration
  name: (simple_identifier) @name) @class

;; Interface definitions
(interface_declaration
  name: (simple_identifier) @name) @interface

;; Object declarations
(object_declaration
  name: (simple_identifier) @name) @class

;; Function definitions
(function_declaration
  name: (simple_identifier) @name) @function

;; Property declarations
(property_declaration
  name: (simple_identifier) @name) @variable

;; Constant declarations
(property_declaration
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "const"))
  name: (simple_identifier) @name) @variable

;; Type alias declarations
(type_alias
  name: (type_identifier) @name) @type_alias

;; Enum class declarations
(enum_class
  name: (simple_identifier) @name) @enum

;; Companion object declarations
(companion_object
  name: (simple_identifier) @name) @class

;; Secondary constructor declarations
(secondary_constructor
  (constructor_delegation_call
    (constructor_invocation
      (user_type
        (type_identifier) @name)))) @function 