;; Ruby structure query
;; Captures method definitions, class definitions, module definitions, and more

;; Method definitions
(method
  name: (identifier) @name) @method

;; Class definitions
(class
  name: (constant) @name) @class

;; Module definitions
(module
  name: (constant) @name) @module

;; Singleton method definitions
(singleton_method
  name: (identifier) @name) @singleton_method

;; Constant declarations
(constant
  name: (constant) @name) @constant

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