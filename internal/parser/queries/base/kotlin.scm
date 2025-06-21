;; Kotlin structure query
;; Captures class definitions, function definitions, variable definitions, and more

(package_header
  (qualified_identifier) @package.full_name
  ) @package

(import
  (qualified_identifier) @import.full_name
  ) @import

;; Class definitions
(class_definition
  name: (identifier) @name) @definition.class

;; Object definitions
(object_definition
  name: (identifier) @name) @definition.object

;; Function definitions
(function_definition
  name: (identifier) @name) @definition.function

;; Property definitions
(property_definition
  (identifier) @name) @definition.property


;; Type alias definitions
(type_alias
  (identifier) @name) @definition.type_alias

;; Enum class definitions
(enum_entry
  (identifier) @name) @definition.enum

;; Companion object definitions
(companion_object
  name: (identifier) @name) @definition.companion

;; Constructor definitions
(class_definition
  name: (identifier) @name) @definition.constructor

(call_expression

  ) @call.method