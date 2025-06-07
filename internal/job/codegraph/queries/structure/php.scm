;; PHP structure query
;; Captures function definitions, class definitions, interface definitions, and more

;; Function definitions
(function_definition
  name: (name) @name) @function_definition

;; Method declarations
(method_declaration
  name: (name) @name) @method_declaration

;; Class declarations
(class_declaration
  name: (name) @name) @class_declaration

;; Interface declarations
(interface_declaration
  name: (name) @name) @interface_declaration

;; Trait declarations
(trait_declaration
  name: (name) @name) @trait_declaration

;; Namespace definitions
(namespace_definition
  name: (name) @name) @namespace_definition

;; Property declarations
(property_declaration
  (property_element
    name: (variable_name) @name)) @property_declaration

;; Constant declarations
(const_declaration
  (const_element
    name: (name) @name)) @const_declaration

;; Variable declarations
(variable_declaration
  (variable_name) @name) @variable_declaration

;; Type alias declarations (using)
(use_declaration
  (name) @name) @type_alias

;; Enum declarations (PHP 8.1+)
(enum_declaration
  name: (name) @name) @enum 