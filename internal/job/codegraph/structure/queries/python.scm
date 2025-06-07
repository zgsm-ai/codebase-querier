;; Python structure query
;; Captures function definitions, class definitions, variable declarations, and more

;; Function definitions
(function_definition
  name: (identifier) @name) @function_definition

;; Class definitions
(class_definition
  name: (identifier) @name) @class_definition

;; Decorated definitions
(decorated_definition
  definition: (function_definition
    name: (identifier) @name)) @decorated_definition

;; Variable assignments
(assignment
  left: (identifier) @name) @assignment

;; Constant assignments (uppercase)
(assignment
  left: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @assignment

;; Method definitions (inside classes)
(class_definition
  body: (block
    (function_definition
      name: (identifier) @name))) @function

;; Type aliases (Python 3.12+)
(type_alias
  name: (identifier) @name) @type_alias

;; Enum definitions (Python 3.4+)
(call
  function: (identifier) @enum_name
  arguments: (argument_list
    (identifier) @name)
  (#eq? @enum_name "Enum")) @enum 