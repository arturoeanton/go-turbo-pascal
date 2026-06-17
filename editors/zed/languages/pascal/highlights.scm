; Syntax highlighting queries for Pascal (tree-sitter-pascal).
; Node names follow the Isopod/tree-sitter-pascal grammar; adjust if you pin a
; different grammar in extension.toml.

(comment) @comment

(str) @string
(char) @string

[
  (intNum)
  (hexNum)
  (realNum)
] @number

[
  "program" "unit" "interface" "implementation" "uses"
  "begin" "end" "var" "const" "type" "array" "record" "set" "of"
  "procedure" "function" "constructor" "destructor"
  "object" "class" "virtual" "override" "inherited"
  "if" "then" "else" "case" "for" "while" "repeat" "until" "do"
  "to" "downto" "with" "goto" "label" "nil"
  "and" "or" "not" "xor" "div" "mod" "shl" "shr" "in"
] @keyword

(declType) @type
(typeref) @type

(identifier) @variable
