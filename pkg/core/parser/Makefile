yaccgen:
	goyacc -o gram.go -p yy gram.y
gen:
	ragel -Z -G2 -o lex.go lex.rl

build: gen yaccgen