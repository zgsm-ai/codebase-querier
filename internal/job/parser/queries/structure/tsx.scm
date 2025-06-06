;; TSX structure query
;; Captures function definitions, class definitions, interfaces, types, JSX elements, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function

;; Function expressions
(variable_declarator
  name: (identifier) @name
  value: (function)) @function

;; Arrow functions
(variable_declarator
  name: (identifier) @name
  value: (arrow_function)) @function

;; Class declarations
(class_declaration
  name: (identifier) @name) @class

;; Class expressions
(variable_declarator
  name: (identifier) @name
  value: (class)) @class

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @function

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @type_alias

;; Type declarations
(type_declaration
  name: (type_identifier) @name) @type_alias

;; Enum declarations
(enum_declaration
  name: (type_identifier) @name) @enum

;; Namespace declarations
(namespace_declaration
  name: (identifier) @name) @class

;; Module declarations
(module_declaration
  name: (identifier) @name) @class

;; Variable declarations
(variable_declarator
  name: (identifier) @name) @variable

;; Constant declarations
(variable_declarator
  name: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable

;; Generic type declarations
(type_parameter_declaration
  name: (type_identifier) @name) @type_alias

;; Decorator declarations
(decorator
  expression: (call_expression
    function: (identifier) @name)) @function

;; Abstract class declarations
(class_declaration
  name: (identifier) @name
  (#has-ancestor? @name abstract_class_declaration)) @class

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  (#has-ancestor? @name abstract_method_declaration)) @function

;; Mapped type declarations
(mapped_type_clause
  name: (type_identifier) @name) @type_alias

;; Conditional type declarations
(conditional_type
  name: (type_identifier) @name) @type_alias

;; Import type declarations
(import_type
  name: (type_identifier) @name) @type_alias

;; Export type declarations
(export_type
  name: (type_identifier) @name) @type_alias

;; JSX Function Components
(export_statement
  (variable_declaration
    (variable_declarator
      name: (identifier) @name
      value: (arrow_function
        body: (jsx_element))))) @function

;; JSX Class Components
(class_declaration
  name: (identifier) @name
  body: (class_body
    (method_definition
      name: (property_identifier) @render
      (#eq? @render "render")
      body: (statement_block
        (return_statement
          (jsx_element)))))) @class

;; JSX Element declarations (custom components)
(jsx_element
  open_tag: (jsx_opening_element
    name: (identifier) @name)) @class

;; JSX Self-closing elements
(jsx_self_closing_element
  name: (identifier) @name) @class

;; JSX Fragment declarations
(jsx_fragment) @class

;; JSX Namespace components
(jsx_element
  open_tag: (jsx_opening_element
    name: (member_expression
      object: (identifier) @namespace
      property: (property_identifier) @name))) @class

;; JSX Props interface declarations
(interface_declaration
  name: (type_identifier) @name
  (#match? @name "^.*Props$")) @interface

;; React Hook declarations
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (call_expression
      function: (identifier) @hook
      (#match? @hook "^use[A-Z]")))) @function

;; React Context declarations
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (call_expression
      function: (member_expression
        object: (identifier) @react
        property: (property_identifier) @create
        (#eq? @react "React")
        (#eq? @create "createContext"))))) @class

;; React Component type declarations
(type_alias_declaration
  name: (type_identifier) @name
  value: (union_type
    (type_identifier) @react
    (#eq? @react "React"))) @type_alias

;; React Event handler declarations
(method_definition
  name: (property_identifier) @name
  (#match? @name "^handle.*$")) @function

;; React Lifecycle method declarations
(method_definition
  name: (property_identifier) @name
  (#match? @name "^(componentDidMount|componentDidUpdate|componentWillUnmount|getDerivedStateFromProps|shouldComponentUpdate|getSnapshotBeforeUpdate|componentDidCatch|componentWillMount|componentWillReceiveProps|componentWillUpdate|UNSAFE_componentWillMount|UNSAFE_componentWillReceiveProps|UNSAFE_componentWillUpdate)$")) @function 