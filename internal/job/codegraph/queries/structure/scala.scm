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
  name: (identifier) @name) @class_definition

;; Object declarations
(object_definition
  name: (identifier) @name) @object_definition

;; Trait definitions
(trait_definition
  name: (identifier) @name) @trait_definition

;; Method definitions
(function_definition
  name: (identifier) @name) @def

;; Type alias definitions
(type_definition
  name: (identifier) @name) @type_definition

;; Enum declarations
(enum_case
  name: (identifier) @name) @enum

;; Companion object declarations
(object_definition
  (modifiers
    (modifier "companion"))?
  name: (identifier) @name) @class_definition

;; Case class definitions
(class_definition
  (modifiers
    (modifier "case"))?
  name: (identifier) @name) @case_class

;; Implicit class definitions
(class_definition
  (modifiers
    (modifier "implicit"))?
  name: (identifier) @name) @class_definition

;; Package object declarations
(package_object
  name: (identifier) @name) @class_definition

;; Value definitions (variables)
(val_definition
  name: (identifier) @name) @val_definition

;; Variable definitions
(var_definition
  name: (identifier) @name) @var_definition 