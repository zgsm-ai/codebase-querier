(import_statement ) @import @name
(import_from_statement )  @import_from @name
;; Python structure query
;; Captures function definitions, class definitions, variable declarations, and more

;; Function definitions
(function_definition
  name: (identifier) @name) @function

;; Class definitions
(class_definition
  name: (identifier) @name) @class

;; Decorated functions
(decorated_definition
  definition: (function_definition
                name: (identifier) @name)) @function

;; Variable assignments
(assignment
  left: (identifier) @name) @variable

;; Constant assignments (uppercase)
(assignment
  left: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @constant

;; Method definitions (inside classes)
(class_definition
  body: (block
          (function_definition
            name: (identifier) @name))) @method

;; Type aliases
(assignment
  left: (identifier) @name
  right: (call
           function: (identifier)
           (#eq? @name "TypeVar"))) @type

;; Enum definitions (Python 3.4+)
(class_definition
  name: (identifier) @name
  superclasses: (argument_list
                  (identifier) @base
                  (#eq? @base "Enum"))) @enum

;; Dataclass definitions
(decorated_definition
  (decorator
    (expression (identifier) @decorator)
    (#eq? @decorator "dataclass"))
  definition: (class_definition
                name: (identifier) @name)) @dataclass

;; Protocol definitions
(class_definition
  name: (identifier) @name
  superclasses: (argument_list
                  (identifier) @base
                  (#eq? @base "Protocol"))
  ) @protocol