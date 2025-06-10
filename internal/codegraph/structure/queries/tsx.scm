(lexical_declaration (variable_declarator name: (identifier) @name ) ) @lexical_declaration

;; Function declarations
(function_declaration
  name: (identifier) @name) @function_declaration

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name )
  ) @variable_declaration


;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @method_definition

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @interface_declaration

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @type_alias_declaration


;; Enum declarations
(enum_declaration
  name: (identifier) @name) @enum_declaration


;; Decorator declarations
(decorator
  (identifier) @name) @decorator

;; Abstract class declarations
(class_declaration
  name: (type_identifier) @name ) @class_declaration

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  ) @method_definition


;; Conditional type declarations
(conditional_type
  left: (type) @name
  ) @conditional_type

;; Import type declarations
(import_statement ) @import_type @name

;; Export type declarations
(export_statement ) @export_type @name



;; JSX Element declarations (custom components)
(jsx_element
  open_tag: (jsx_opening_element
              name: (identifier) @name)) @jsx_element

;; JSX Self-closing elements
(jsx_self_closing_element
  name: (identifier) @name) @jsx_self_closing_element



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


;; React Component type declarations
(type_alias_declaration
  name: (type_identifier) @name
  value: (union_type
           (type_identifier) @react
           (#eq? @react "React"))) @type_alias_declaration