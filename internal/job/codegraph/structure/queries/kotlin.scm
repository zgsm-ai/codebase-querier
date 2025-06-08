;; Kotlin structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Class declarations
(class_declaration
  name: (type_identifier) @name) @class

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface

;; Object declarations
(object_declaration
  name: (type_identifier) @name) @object

;; Function declarations
(function_declaration
  name: (identifier) @name) @function

;; Property declarations
(property_declaration
  name: (identifier) @name) @property

;; Constant declarations
(property_declaration
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "const"))
  name: (identifier) @name) @constant

;; Type alias declarations
(type_alias
  name: (type_identifier) @name) @type

;; Enum class declarations
(enum_class
  name: (type_identifier) @name) @enum

;; Companion object declarations
(companion_object
  (object_declaration 
    name: (type_identifier) @name)) @companion

;; Constructor declarations
(class_declaration
  (primary_constructor
    (class_parameter
      name: (identifier) @name))) @constructor