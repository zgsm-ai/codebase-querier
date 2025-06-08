;; TSX structure query
;; Captures function definitions, class definitions, interfaces, types, JSX elements, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @function_declaration

;; Function expressions
(variable_declarator
  name: (identifier) @name
  value: (function_expression)) @variable_declarator  ;; 修正: function → function_expression

;; Arrow functions
(variable_declarator
  name: (identifier) @name
  value: (arrow_function)) @variable_declarator

;; Class declarations
(class_declaration
  name: (identifier) @name) @class_declaration

;; Class expressions
(variable_declarator
  name: (identifier) @name
  value: (class_expression)) @variable_declarator  ;; 修正: class → class_expression

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @method_definition

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface_declaration

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @type_alias_declaration

;; Type declarations（建议统一使用 type_alias_declaration）
(type_alias_declaration
  name: (type_identifier) @name) @type_declaration  ;; 修正: 统一使用 type_alias_declaration

;; Enum declarations
(enum_declaration
  name: (type_identifier) @name) @enum_declaration

;; Namespace declarations
(namespace_declaration
  name: (identifier) @name) @namespace_declaration

;; Module declarations
(module_declaration
  name: (identifier) @name) @module_declaration

;; Variable declarations
(variable_declarator
  name: (identifier) @name) @variable_declarator

;; Constant declarations
(variable_declarator
  name: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @variable_declarator

;; Generic type declarations
(type_parameter_declaration
  name: (type_identifier) @name) @type_parameter_declaration

;; Decorator declarations
(decorator
  expression: (call_expression
    function: (identifier) @name)) @decorator

;; Abstract class declarations
(class_declaration
  name: (identifier) @name
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "abstract"))) @class_declaration  ;; 修正: 直接匹配 abstract 修饰符

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  (modifiers
    (modifier) @modifier
    (#eq? @modifier "abstract"))) @method_definition  ;; 修正: 直接匹配 abstract 修饰符

;; Mapped type declarations
(mapped_type_clause
  name: (type_identifier) @name) @mapped_type_clause

;; Conditional type declarations
(conditional_type
  name: (type_identifier) @name) @conditional_type

;; Import type declarations
(import_type
  name: (type_identifier) @name) @import_type

;; Export type declarations
(export_type
  name: (type_identifier) @name) @export_type

;; JSX Function Components
(export_statement
  (variable_declaration
    (variable_declarator
      name: (identifier) @name
      value: (arrow_function
        body: (jsx_element))))) @export_statement

;; JSX Class Components
(class_declaration
  name: (identifier) @name
  body: (class_body
    (method_definition
      name: (property_identifier) @render
      (#eq? @render "render")
      body: (statement_block
        (return_statement
          (jsx_element)))))) @class_declaration

;; JSX Element declarations (custom components)
(jsx_element
  open_tag: (jsx_opening_element
    name: (identifier) @name)) @jsx_element

;; JSX Self-closing elements
(jsx_self_closing_element
  name: (identifier) @name) @jsx_self_closing_element

;; JSX Fragment declarations
(jsx_fragment) @jsx_fragment

;; JSX Namespace components
(jsx_element
  open_tag: (jsx_opening_element
    name: (member_expression
      object: (identifier) @namespace
      property: (property_identifier) @name))) @jsx_element

;; JSX Props interface declarations
(interface_declaration
  name: (type_identifier) @name
  (#match? @name "^.*Props$")) @interface_declaration

;; React Hook declarations
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (call_expression
      function: (identifier) @hook
      (#match? @hook "^use[A-Z]")))) @variable_declaration

;; React Context declarations
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (call_expression
      function: (member_expression
        object: (identifier) @react
        property: (property_identifier) @create
        (#eq? @react "React")
        (#eq? @create "createContext"))))) @variable_declaration  ;; 修正: @class_declaration → @variable_declaration

;; React Component type declarations
(type_alias_declaration
  name: (type_identifier) @name
  value: (union_type
    (type_identifier) @react
    (#eq? @react "React"))) @type_alias_declaration

;; React Event handler declarations
(method_definition
  name: (property_identifier) @name
  (#match? @name "^handle.*$")) @method_definition

;; React Lifecycle method declarations
(method_definition
  name: (property_identifier) @name
  (#match? @name "^(componentDidMount|componentDidUpdate|componentWillUnmount|getDerivedStateFromProps|shouldComponentUpdate|getSnapshotBeforeUpdate|componentDidCatch|componentWillMount|componentWillReceiveProps|componentWillUpdate|UNSAFE_componentWillMount|UNSAFE_componentWillReceiveProps|UNSAFE_componentWillUpdate)$")) @method_definition