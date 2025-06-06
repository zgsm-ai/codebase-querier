;; Python structure query
;; Captures function definitions, class definitions, and variable assignments

;; Function definitions
(function_definition
  name: (identifier) @name) @function

;; Class definitions
(class_definition
  name: (identifier) @name) @class

;; Method definitions (inside classes)
(class_definition
  body: (block
    (function_definition
      name: (identifier) @name))) @function

;; Type aliases (Python 3.12+)
(type_alias
  name: (identifier) @name) @type_alias

;; Variable assignments
(assignment
  left: (identifier) @name) @variable

;; Constant assignments (convention: uppercase names)
(assignment
  left: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable

;; Enum definitions (Python 3.4+)
(call
  function: (identifier) @enum_name
  arguments: (argument_list
    (identifier) @name)
  (#eq? @enum_name "Enum")) @enum 