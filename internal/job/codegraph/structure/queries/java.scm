;; Java structure query
;; Captures class definitions, interface definitions, method definitions, and more

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @interface

;; Method declarations
(method_declaration
  name: (identifier) @name) @method

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @constructor

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @enum

;; Field declarations
(field_declaration
  declarator: (variable_declarator
    name: (identifier) @name)) @field

;; Constant field declarations (static final)
(field_declaration
  (modifiers
    "static"
    "final")
  (variable_declarator
    name: (identifier) @name)) @constant

;; Enum constants
(enum_constant
  name: (identifier) @name) @enum_constant

;; Type parameters
(type_parameters
  (type_parameter) @type_parameter)

;; Annotation declarations
(annotation_type_declaration
  name: (identifier) @name) @annotation

;; Record declarations (Java 14+)
(record_declaration
  name: (identifier) @name) @record