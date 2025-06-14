(function_declaration
  name: (identifier) @name) @declaration.function

(import_declaration (import_spec_list (import_spec) @name )*  (import_spec )* @name ) @declaration.import

(method_declaration
  name: (field_identifier) @name) @declaration.method

(package_clause "package" (package_identifier) @name) @package

(type_declaration (type_spec name: (type_identifier) @name type: (interface_type))) @declaration.interface

(type_declaration (type_spec name: (type_identifier) @name type: (struct_type))) @declaration.struct

(type_declaration (type_spec name: (type_identifier) @name type: (type_identifier))) @declaration.type_alias


(var_declaration (var_spec name: (identifier) @name)) @declaration.var

(const_declaration (const_spec name: (identifier) @name)) @declaration.const