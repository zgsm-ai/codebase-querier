(function_declaration
  name: (identifier) @name) @definition.function

(import_declaration (import_spec_list (import_spec) @name )*  (import_spec )* @name ) @import

(method_declaration
  name: (field_identifier) @name) @definition.method

(package_clause "package" (package_identifier) @name) @package

(type_declaration (type_spec name: (type_identifier) @name type: (interface_type))) @definition.interface

(type_declaration (type_spec name: (type_identifier) @name type: (struct_type))) @definition.struct

(type_declaration (type_spec name: (type_identifier) @name type: (type_identifier))) @definition.type_alias


(var_declaration (var_spec name: (identifier) @name)) @definition.var

(const_declaration (const_spec name: (identifier) @name)) @difinition.const