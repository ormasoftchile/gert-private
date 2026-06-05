package gxl

// Position identifies a 1-based source location in a GXL expression.
type Position struct {
	Line   int
	Column int
}

// Node is implemented by every GXL AST node.
type Node interface {
	Pos() Position
	node()
}

// LiteralKind identifies the literal production matched by a LiteralNode.
type LiteralKind string

const (
	LiteralString LiteralKind = "string"
	LiteralNumber LiteralKind = "number"
	LiteralBool   LiteralKind = "bool"
	LiteralNull   LiteralKind = "null"
)

// LiteralNode represents a string, number, bool, or null literal.
type LiteralNode struct {
	Position Position
	Kind     LiteralKind
	Lexeme   string
	Value    string
}

func (n *LiteralNode) Pos() Position { return n.Position }
func (*LiteralNode) node()           {}

// UnaryNode represents prefix not or unary minus.
type UnaryNode struct {
	Position Position
	Op       TokenKind
	Expr     Node
}

func (n *UnaryNode) Pos() Position { return n.Position }
func (*UnaryNode) node()           {}

// BinaryNode represents a left-associative binary expression.
type BinaryNode struct {
	Position Position
	Op       TokenKind
	Left     Node
	Right    Node
}

func (n *BinaryNode) Pos() Position { return n.Position }
func (*BinaryNode) node()           {}

// PathSegmentKind identifies GDP path segment syntax.
type PathSegmentKind string

const (
	PathSegmentField PathSegmentKind = "field"
	PathSegmentIndex PathSegmentKind = "index"
)

// PathSegment is one GDP field or array-index segment after the root.
type PathSegment struct {
	Kind     PathSegmentKind
	Name     string
	Index    string
	Position Position
}

// PathNode represents a GERT Dotted Path variable access.
type PathNode struct {
	Position Position
	Root     string
	Segments []PathSegment
}

func (n *PathNode) Pos() Position { return n.Position }
func (*PathNode) node()           {}

// CallNode represents a top-level builtin call or namespaced stdlib call.
type CallNode struct {
	Position  Position
	Namespace string
	Method    string
	Args      []Node
}

func (n *CallNode) Pos() Position { return n.Position }
func (*CallNode) node()           {}
