(import_statement
  (import_clause
    ) @import.names
  ) @import

(import_statement
  source: (string
            ) @import.name
  ) @import

;; Function declarations
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (formal_parameters) @definition.function.parameters

  ) @definition.function

;; 全局变量
(program
  (_
    (variable_declarator) @global_variable
    )
  )


;; 函数、变量

(variable_declarator) @variable



;; Object properties
(pair
  key: (property_identifier) @name) @declaration.property

;; Export declarations
(export_statement
  declaration: (function_declaration
                 name: (identifier) @name)) @declaration.export_function

;; Export named declarations
(export_statement
  (export_clause
    (export_specifier
      name: (identifier) @name))) @declaration.export_statement

;; 函数调用
(call_expression
  function: (member_expression
              object: (identifier) @call.function.object
              property: (property_identifier) @call.function.name
              )
  arguments: (arguments) @call.function.arguments
  ) @call.function