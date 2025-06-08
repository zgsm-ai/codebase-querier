;; C# structure query
;; Captures method definitions, class definitions, interface definitions, and more

;; Method declarations
(method_declaration
  name: (identifier) @name) @method

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @interface

;; Struct declarations
(struct_declaration
  name: (identifier) @name) @struct

;; Property declarations
(property_declaration
  name: (identifier) @name) @property

;; Delegate declarations
(delegate_declaration
  name: (identifier) @name) @delegate

;; Event declarations
(event_declaration
  name: (identifier) @name) @event

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @constructor

;; Destructor declarations
(destructor_declaration
  name: (identifier) @name) @destructor

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @enum

;; Field declarations
(field_declaration
  (variable_declaration
    (variable_declarator
      name: (identifier) @name))) @field

;; Constant declarations
(field_declaration
  modifier_list: (modifier_list
    (modifier) @modifier
    (#eq? @modifier "const"))
  (variable_declaration
    (variable_declarator
      name: (identifier) @name))) @constant

;; Indexer declarations
(indexer_declaration
  name: (identifier) @name) @indexer

;; Operator declarations
(operator_declaration
  name: (identifier) @name) @operator

;; Type parameter declarations
(type_parameter
  name: (identifier) @name) @type_parameter

;; Record definitions (C# 9.0+)
(record_declaration
  name: (identifier) @name) @record

;; Record struct definitions (C# 10.0+)
(record_struct_declaration
  name: (identifier) @name) @record_struct

;; Local function definitions
(local_function_statement
  name: (identifier) @name) @local_function

;; Conversion operator definitions
(conversion_operator_declaration
  type: (identifier) @name) @conversion_operator