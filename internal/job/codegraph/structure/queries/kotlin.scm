;; Kotlin structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Object declarations
(object_declaration
  name: (identifier) @name) @object

;; Function declarations
(function_declaration
  name: (identifier) @name) @function

;; Property declarations
(property_declaration
   (identifier) @name) @property


;; Type alias declarations
(type_alias
   (identifier) @name) @type

;; Enum class declarations
(enum_entry
  (identifier) @name) @enum

;; Companion object declarations
(companion_object
    name: (identifier) @name) @companion

;; Constructor declarations
(class_declaration
    name: (identifier) @name) @constructor