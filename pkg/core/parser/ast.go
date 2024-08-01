package parser

type Node interface {
	iNode()
}

type SayHelloCommand struct {
	Node
}
