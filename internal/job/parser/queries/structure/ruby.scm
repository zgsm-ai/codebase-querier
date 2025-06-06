;; Ruby structure query
;; Captures class definitions, module definitions, method definitions, and more

;; Class definitions
(class
  name: (constant) @name) @class

;; Module definitions
(module
  name: (constant) @name) @class

;; Method definitions
(method
  name: (identifier) @name) @function

;; Singleton method definitions
(singleton_method
  name: (identifier) @name) @function

;; Constant assignments
(assignment
  left: (constant) @name) @variable

;; Constant assignments (convention: uppercase names)
(assignment
  left: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable

;; Module method definitions (inside modules)
(module
  body: (body_statement
    (method
      name: (identifier) @name))) @function

;; Class method definitions (inside classes)
(class
  body: (body_statement
    (singleton_method
      name: (identifier) @name))) @function

;; Instance method definitions (inside classes)
(class
  body: (body_statement
    (method
      name: (identifier) @name))) @function 