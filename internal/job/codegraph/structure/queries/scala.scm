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
  name: (identifier) @name) @class

;; Object declarations
(object_definition
  name: (identifier) @name) @object

;; Trait definitions
(trait_definition
  name: (identifier) @name) @trait

;; Method definitions
(function_definition
  name: (identifier) @name) @method

;; Type alias definitions
(type_definition
  name: (identifier) @name) @type

;; Enum definitions (Scala 3)
(enum_definition
  name: (identifier) @name) @enum

;; Case class definitions
(class_definition
  modifier_list: (modifier_list
    (identifier) @modifier
    (#eq? @modifier "case"))
  name: (identifier) @name) @case_class

;; Value definitions (val)
(val_definition
  pattern: (identifier) @name) @val

;; Variable definitions (var)
(var_definition
  pattern: (identifier) @name) @var

;; Package object definitions
(package_object_definition
  name: (identifier) @name) @package_object

;; Implicit class definitions
(class_definition
  modifier_list: (modifier_list
    (identifier) @modifier
    (#eq? @modifier "implicit"))
  name: (identifier) @name) @implicit_class 