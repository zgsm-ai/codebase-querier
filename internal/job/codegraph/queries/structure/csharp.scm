;; C# structure query
;; Captures method definitions, class definitions, interface definitions, and more

;; Method declarations
(method_declaration
  name: (identifier) @name) @method_declaration

;; Class declarations
(class_declaration
  name: (identifier) @name) @class_declaration

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @interface_declaration

;; Struct declarations
(struct_declaration
  name: (identifier) @name) @struct_declaration

;; Property declarations
(property_declaration
  name: (identifier) @name) @property_declaration

;; Delegate declarations
(delegate_declaration
  name: (identifier) @name) @delegate_declaration

;; Event declarations
(event_declaration
  name: (identifier) @name) @event_declaration

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @constructor_declaration

;; Destructor declarations
(destructor_declaration
  name: (identifier) @name) @destructor_declaration

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @enum_declaration

;; Field declarations
(field_declaration
  declarator: (variable_declarator
    name: (identifier) @name)) @field_declaration

;; Constant declarations
(field_declaration
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "const"))
  declarator: (variable_declarator
    name: (identifier) @name)) @field_declaration

;; Indexer declarations
(indexer_declaration
  name: (identifier) @name) @indexer_declaration

;; Operator declarations
(operator_declaration
  name: (identifier) @name) @operator_declaration

;; Type parameter declarations
(type_parameter
  name: (identifier) @name) @type_parameter

;; Record definitions (C# 9.0+)
(record_declaration
  name: (identifier) @name) @class

;; Record struct definitions (C# 10.0+)
(record_struct_declaration
  name: (identifier) @name) @struct

;; Local function definitions
(local_function_statement
  name: (identifier) @name) @function

;; Conversion operator definitions
(conversion_operator_declaration
  name: (identifier) @name) @function 