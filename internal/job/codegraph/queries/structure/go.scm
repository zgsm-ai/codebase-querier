;; Go structure query
;; Captures function definitions, type definitions, variable declarations, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function_declaration

;; Method declarations
(method_declaration
  name: (field_identifier) @name) @method_declaration

;; Type declarations (struct)
(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (struct_type))) @type_declaration

;; Type declarations (interface)
(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (interface_type))) @type_declaration

;; Variable declarations
(var_declaration
  (var_spec
    name: (identifier) @name)) @var_declaration

;; Constant declarations
(const_declaration
  (const_spec
    name: (identifier) @name)) @const_declaration