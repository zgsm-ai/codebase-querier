;; C# structure query
;; Captures various C# code structures including:
;; - Class definitions
;; - Struct definitions
;; - Interface definitions
;; - Method definitions
;; - Property definitions
;; - Field definitions
;; - Event definitions
;; - Delegate definitions
;; - Enum definitions

;; Class definitions
(class_declaration
  name: (identifier) @name) @class

;; Struct definitions
(struct_declaration
  name: (identifier) @name) @struct

;; Interface definitions
(interface_declaration
  name: (identifier) @name) @interface

;; Method definitions
(method_declaration
  name: (identifier) @name) @function

;; Property definitions
(property_declaration
  name: (identifier) @name) @variable

;; Field definitions
(field_declaration
  (variable_declaration
    name: (identifier) @name)) @variable

;; Event definitions
(event_declaration
  name: (identifier) @name) @variable

;; Delegate definitions
(delegate_declaration
  name: (identifier) @name) @type_alias

;; Enum definitions
(enum_declaration
  name: (identifier) @name) @enum

;; Record definitions (C# 9.0+)
(record_declaration
  name: (identifier) @name) @class

;; Record struct definitions (C# 10.0+)
(record_struct_declaration
  name: (identifier) @name) @struct

;; Local function definitions
(local_function_statement
  name: (identifier) @name) @function

;; Indexer definitions
(indexer_declaration
  name: (identifier) @name) @variable

;; Operator definitions
(operator_declaration
  name: (identifier) @name) @function

;; Conversion operator definitions
(conversion_operator_declaration
  name: (identifier) @name) @function

;; Constructor definitions
(constructor_declaration
  name: (identifier) @name) @function

;; Destructor definitions
(destructor_declaration
  name: (identifier) @name) @function 