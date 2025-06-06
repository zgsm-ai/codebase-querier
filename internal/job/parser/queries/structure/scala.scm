;; Scala structure query
;; Captures various Scala code structures including:
;; - Class definitions
;; - Object declarations
;; - Trait definitions
;; - Method definitions
;; - Type aliases
;; - Enum declarations

;; Class definitions
(class_definition
  name: (type_identifier) @name) @class

;; Object declarations
(object_definition
  name: (type_identifier) @name) @class

;; Trait definitions
(trait_definition
  name: (type_identifier) @name) @interface

;; Method definitions
(function_definition
  name: (identifier) @name) @function

;; Type alias definitions
(type_definition
  name: (type_identifier) @name) @type_alias

;; Enum declarations
(enum_case
  name: (identifier) @name) @enum

;; Companion object declarations
(object_definition
  (modifiers
    (modifier "companion"))?
  name: (type_identifier) @name) @class

;; Case class definitions
(class_definition
  (modifiers
    (modifier "case"))?
  name: (type_identifier) @name) @class

;; Implicit class definitions
(class_definition
  (modifiers
    (modifier "implicit"))?
  name: (type_identifier) @name) @class

;; Package object declarations
(package_object
  name: (identifier) @name) @class

;; Value definitions (variables)
(val_definition
  pattern: (identifier) @name) @variable

;; Variable definitions
(var_definition
  pattern: (identifier) @name) @variable 