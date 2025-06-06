(function_declaration
  name: (identifier) @name) @function

(method_declaration
  name: (field_identifier) @name
  parameters: (parameter_list) @params
  result: (_)? @result) @function

(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (struct_type))) @struct

(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (interface_type))) @interface

(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (_) @typekind
      (#not-match? @typekind "struct_type")
      (#not-match? @typekind "interface_type"))) @type_alias

(var_declaration
  (var_spec
    name: (identifier) @name)) @variable

(const_declaration
  (const_spec
    name: (identifier) @name)) @variable