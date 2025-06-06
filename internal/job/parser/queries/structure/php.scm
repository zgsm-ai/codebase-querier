;; PHP structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Class definitions
(class_declaration
  name: (name) @name) @class

;; Interface definitions
(interface_declaration
  name: (name) @name) @interface

;; Trait definitions
(trait_declaration
  name: (name) @name) @class

;; Function definitions
(function_definition
  name: (name) @name) @function

;; Method definitions (inside classes)
(method_declaration
  name: (name) @name) @function

;; Constant declarations
(const_declaration
  name: (name) @name) @variable

;; Property declarations (inside classes)
(property_declaration
  (property_element
    (variable_name) @name)) @variable

;; Namespace declarations
(namespace_definition
  name: (name) @name) @class

;; Type alias declarations (using)
(use_declaration
  (name) @name) @type_alias

;; Enum declarations (PHP 8.1+)
(enum_declaration
  name: (name) @name) @enum 